package ssh

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/benschlueter/delegatio/cli/kubernetes"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// TODO: Maybe we should wait for all goroutines to finish before we return. Might require heavy refactoring.

type sshRelay struct {
	log    *zap.Logger
	client *kubernetes.KubernetesClient
}

// NewSSHRelay returns a sshRelay.
func NewSSHRelay(client *kubernetes.KubernetesClient, log *zap.Logger) *sshRelay {
	return &sshRelay{
		client: client,
		log:    log,
	}
}

func (s *sshRelay) StartServer(ctx context.Context) {
	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn

	config := &ssh.ServerConfig{
		// Define a function to run when a client attempts a password login
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in a production setting.
			if c.User() == "foo" && string(pass) == "bar" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		// You may also explicitly allow anonymous client authentication, though anon bash
		// sessions may not be a wise idea
		// NoClientAuth: true,
	}

	// You can generate a keypair with 'ssh-keygen -t rsa'
	privateBytes, err := os.ReadFile("./server_test")
	if err != nil {
		log.Fatal("Failed to load private key (./server_test)", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		log.Fatalf("Failed to listen on 2200 (%s)", err)
	}
	defer listener.Close()

	// Accept all connections
	s.log.Info("Listening on  `0.0.0.0:2200`")
	go func(ctx context.Context) {
		for {

			tcpConn, err := listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				s.log.Error("failed to accept incoming connection", zap.Error(err))
				continue
			}
			s.log.Info("handling incomming connection", zap.String("addr", tcpConn.RemoteAddr().String()))
			go s.handeConn(ctx, tcpConn, config)
		}
	}(ctx)
	<-ctx.Done()
}

func (s *sshRelay) handeConn(ctx context.Context, tcpConn net.Conn, config *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming net.Conn.
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		s.log.Info("failed to handshake", zap.Error(err))
		return
	}

	s.log.Info("new ssh connection", zap.String("addr", sshConn.RemoteAddr().String()), zap.Binary("client version", sshConn.ClientVersion()))
	// Discard all global out-of-band Requests
	go ssh.DiscardRequests(reqs)
	// Accept all channels
	go s.handleChannels(ctx, chans)
}

func (s *sshRelay) handleChannels(ctx context.Context, chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go s.handleChannel(ctx, newChannel)
	}
}

func (s *sshRelay) handleChannel(ctx context.Context, newChannel ssh.NewChannel) {
	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	if t := newChannel.ChannelType(); t != "session" {
		s.log.Error("unknown channel type", zap.String("type", newChannel.ChannelType()))
		err := newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		if err != nil {
			s.log.Error("failed to reject channel", zap.Error(err))
		}
		return
	}

	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		s.log.Error("could not accept the channel", zap.Error(err))
		return
	}
	defer connection.Close()

	window := &Winsize{
		Queue: make(chan *remotecommand.TerminalSize),
	}

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) != 0 {
					continue
				}
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to `shell` request", zap.Error(err))
				}
			case "pty-req":
				termLen := req.Payload[3]
				window.Queue <- parseDims(req.Payload[termLen+4:])
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to `pty-req`", zap.Error(err))
				}
			case "window-change":
				s.log.Info("window change request received")
				window.Queue <- parseDims(req.Payload)
			}
		}
	}()
	// Fire up "kubectl exec" for this session
	err = s.client.CreatePodShell(ctx,
		"testchallenge",
		"dummyuser-statefulset-0",
		connection,
		connection,
		connection,
		window)
	if err != nil {
		s.log.Error("createPodShell exited with errorcode", zap.Error(err))
		return
	}
	_, err = connection.Write([]byte(fmt.Sprintf("closing connection %v", err)))
	if err != nil {
		s.log.Info("could not send final message")
	}
}

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) *remotecommand.TerminalSize {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return &remotecommand.TerminalSize{
		Width:  uint16(w),
		Height: uint16(h),
	}
}

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Queue chan *remotecommand.TerminalSize
}

// Next sets the size.
func (w *Winsize) Next() *remotecommand.TerminalSize {
	return <-w.Queue
}

// corrected from https://gist.github.com/jpillora/b480fde82bff51a06238
