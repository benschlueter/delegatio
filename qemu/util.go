package qemu

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/multierr"
	"libvirt.org/go/libvirt"
)

func (l *LibvirtInstance) uploadBaseImage(baseVolume *libvirt.StorageVol) (err error) {
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
	for {
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			break
		}
		num, err := stream.Send(buffer)
		if err != nil {
			return err
		}
		transferredBytes += num

	}
	if transferredBytes < int(fi.Size()) {
		return fmt.Errorf("only send %d out of %d bytes", transferredBytes, fi.Size())
	}
	return nil
}
