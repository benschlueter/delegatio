package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/benschlueter/delegatio/cli/kubernetes"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/remotecommand"
)

// TODO: Maybe we should wait for all goroutines to finish before we return. Might require heavy refactoring.

type sshRelay struct {
	log          *zap.Logger
	client       *kubernetes.Client
	handleConnWG *sync.WaitGroup
	users        map[string]struct{}
	publicKeys   map[string]struct{}
}

func main() {
	logger := zap.NewExample()
	client, err := kubernetes.NewK8sClient("admin.conf", logger.Named("k8sAPI"))
	if err != nil {
		panic(err)
	}
	relay := NewSSHRelay(client, logger)
	relay.StartServer(context.Background())
}

// NewSSHRelay returns a sshRelay.
func NewSSHRelay(client *kubernetes.Client, log *zap.Logger) *sshRelay {
	return &sshRelay{
		client:       client,
		log:          log,
		handleConnWG: &sync.WaitGroup{},
		users: map[string]struct{}{
			"testchallenge": {},
		},
		publicKeys: map[string]struct{}{
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLYDO+DPlwJTKYU+S9Q1YkgC7lUJgfsq+V6VxmzdP+omp2EmEIEUsB8WFtr3kAgtAQntaCejJ9ITgoLimkoPs7bV1rA7BZZgRTL2sF+F5zJ1uXKNZz1BVeGGDDXHW5X5V/ZIlH5Bl4kNaAWGx/S5PIszkhyNXEkE6GHsSU4dz69rlutjSbwQRFLx8vjgdAxP9+jUbJMh9u5Dg1SrXiMYpzplJWFt/jI13dDlNTrhWW7790xhHur4fiQbhrVzru29BKNQtSywC+3eH2XKTzobK6h7ECS5X75ghemRIDPw32SHbQP7or1xI+MjFCrZsGyZr1L0yBFNkNAsztpWAqE2FZ": {},
		},
	}
}

func (s *sshRelay) StartServer(ctx context.Context) {
	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn

	config := &ssh.ServerConfig{
		// Function is called to determine if the user is allowed to connect with the ssh server
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			s.log.Info("publickeycallback called", zap.String("user", conn.User()), zap.Binary("session", conn.SessionID()))
			if _, ok := s.users[conn.User()]; !ok {
				return nil, fmt.Errorf("user %s not in database", conn.User())
			}
			encodeKey := base64.StdEncoding.EncodeToString(key.Marshal())
			compareKey := fmt.Sprintf("%s %s", key.Type(), encodeKey)
			if _, ok := s.publicKeys[compareKey]; !ok {
				return nil, fmt.Errorf("pubkey %v not in database", compareKey)
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					"authType": "pk",
					"pubKey":   strings.ToLower(ssh.FingerprintSHA256(key)[7:47]),
				},
			}, nil
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
	s.log.Info("Listening on  \"0.0.0.0:2200\"")
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
			s.handleConnWG.Add(1)
			go s.handeConn(ctx, tcpConn, config)
		}
	}(ctx)
	<-ctx.Done()
	s.handleConnWG.Wait()
}

func (s *sshRelay) handeConn(ctx context.Context, tcpConn net.Conn, config *ssh.ServerConfig) {
	defer s.handleConnWG.Done()
	// Before use, a handshake must be performed on the incoming net.Conn.
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		s.log.Info("failed to handshake", zap.Error(err))
		return
	}
	defer sshConn.Close()
	s.log.Info("new ssh connection",
		zap.String("addr", sshConn.RemoteAddr().String()),
		zap.Binary("client version", sshConn.ClientVersion()),
		zap.Binary("session", sshConn.SessionID()),
		zap.String("keyFingerprint", sshConn.Permissions.Extensions["pubKey"]),
	)
	// Discard all global out-of-band Requests
	// Dont care about graceful termination of this routine
	go ssh.DiscardRequests(reqs)
	exists, err := s.client.StatefulSetExists(ctx, sshConn.User(), sshConn.Permissions.Extensions["pubKey"])
	if err != nil {
		s.log.Error("obtaining statefulset status",
			zap.Error(err),
			zap.String("userID", sshConn.Permissions.Extensions["pubKey"]),
			zap.String("namespace", sshConn.User()),
		)
		return
	}
	s.log.Info("statefulSet exists", zap.Bool("val", exists))
	if !exists {
		if err := s.client.CreateStatefulSetForUser(ctx, sshConn.User(), sshConn.Permissions.Extensions["pubKey"]); err != nil {
			s.log.Error("create statefulset",
				zap.Error(err),
				zap.String("userID", sshConn.Permissions.Extensions["pubKey"]),
				zap.String("namespace", sshConn.User()),
			)
			return
		}
		s.log.Info("created statefulSet",
			zap.String("userID", sshConn.Permissions.Extensions["pubKey"]),
			zap.String("namespace", sshConn.User()))
	}
	if err := s.client.WaitForPodRunning(ctx, sshConn.User(), sshConn.Permissions.Extensions["pubKey"], 1*time.Minute); err != nil {
		s.log.Error("wait for pod",
			zap.Error(err),
			zap.String("userID", sshConn.Permissions.Extensions["pubKey"]),
			zap.String("namespace", sshConn.User()),
		)
		return
	}
	// Accept all channels
	s.handleChannels(ctx, chans, sshConn.User(), sshConn.Permissions.Extensions["pubKey"])
	s.log.Info("closing ssh session",
		zap.String("addr", sshConn.RemoteAddr().String()),
		zap.Binary("client version", sshConn.ClientVersion()),
		zap.Binary("session", sshConn.SessionID()),
		zap.String("keyFingerprint", sshConn.Permissions.Extensions["pubKey"]),
	)
}

func (s *sshRelay) handleChannels(ctx context.Context, chans <-chan ssh.NewChannel, namespace, userID string) {
	// Service the incoming Channel channel in go routine
	handleChannelWg := &sync.WaitGroup{}
	defer handleChannelWg.Wait()
	for {
		select {
		case <-ctx.Done():
			return
		case newChannel := <-chans:
			// when we close the "channel" from newChannel.Accept() in s.handleChannel the ssh.NewChannel
			// is closed from the library side as well. Thus it will always send nil. Return in this case.
			if newChannel == nil {
				return
			}
			handleChannelWg.Add(1)
			go s.handleChannel(ctx, handleChannelWg, newChannel, namespace, userID)
		}
	}
}

func (s *sshRelay) handleChannel(ctx context.Context, wg *sync.WaitGroup, newChannel ssh.NewChannel, namespace, userID string) {
	defer wg.Done()

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
	// request for another logical channel
	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.log.Error("could not accept the channel", zap.Error(err))
		return
	}
	defer func(log *zap.Logger) {
		err := channel.Close()
		if err != nil {
			log.Error("error while closing connection", zap.Error(err))
		}
		log.Info("closed channel connection")
	}(s.log)

	window := &Winsize{
		Queue: make(chan *remotecommand.TerminalSize),
	}

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func(<-chan *ssh.Request) {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) != 0 {
					continue
				}
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to \"shell\" request", zap.Error(err))
				}
			case "pty-req":
				termLen := req.Payload[3]
				window.Queue <- parseDims(req.Payload[termLen+4:])
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				if err := req.Reply(true, nil); err != nil {
					s.log.Error("failled to respond to \"pty-req\" request", zap.Error(err))
				}
			case "window-change":
				s.log.Info("window change request received")
				window.Queue <- parseDims(req.Payload)
			}
		}
	}(requests)
	// Fire up "kubectl exec" for this session
	err = s.client.CreatePodShell(ctx,
		namespace,
		fmt.Sprintf("%s-statefulset-0", userID),
		channel,
		channel,
		channel,
		window)
	if err != nil {
		s.log.Error("createPodShell exited with errorcode", zap.Error(err))
		_, _ = channel.Write([]byte(fmt.Sprintf("closing connection, reason: %v", err)))
		return
	}
	_, _ = channel.Write([]byte("graceful termination"))
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
