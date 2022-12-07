package qemu

import (
	"context"
	"fmt"
	"io"
	"os"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

func (l *LibvirtInstance) uploadBaseImage(ctx context.Context, baseVolume *libvirt.StorageVol) (err error) {
	stream, err := l.conn.NewStream(libvirt.STREAM_NONBLOCK)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Free() }()
	file, err := os.Open(l.imagePath)
	if err != nil {
		return fmt.Errorf("error while opening %s: %s", l.imagePath, err)
	}
	defer func() {
		err = multierr.Append(err, file.Close())
	}()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if err := baseVolume.Upload(stream, 0, uint64(fi.Size()), 0); err != nil {
		return err
	}
	transferredBytes := 0
	buffer := make([]byte, 4*1024*1024)

	// Fill the stream with buffer-chunks of the image.
	// Since this can take long we must make this interruptable in case of
	// a context cancellation.
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				return err
			}
			if err == io.EOF {
				break loop
			}
			num, err := stream.Send(buffer)
			if err != nil {
				return err
			}
			transferredBytes += num
		}
	}
	if transferredBytes < int(fi.Size()) {
		return fmt.Errorf("only send %d out of %d bytes", transferredBytes, fi.Size())
	}
	l.log.Info("image upload successful", zap.Int64("image size", fi.Size()), zap.Int("transferred bytes", transferredBytes))
	return nil
}
