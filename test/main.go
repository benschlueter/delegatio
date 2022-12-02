package main

import (
	"flag"
	"os"

	"github.com/benschlueter/delegatio/test/qemu"
	"github.com/benschlueter/delegatio/test/qemu/definitions"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

func main() {
	var imageLocation string
	var imageType string
	flag.StringVar(&imageLocation, "path", "", "path to the image to measure (required)")
	flag.StringVar(&imageType, "type", "", "type of the image. One of 'qcow2' or 'raw' (required)")
	flag.Parse()
	zapconf := zap.NewDevelopmentConfig()
	log, err := zapconf.Build()
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create logger")
	}

	if imageLocation == "" || imageType == "" {
		flag.Usage()
		os.Exit(1)
	}
	definitions.VolumeBootXMLConfig.BackingStore.Format.Type = imageType

	if err := libvirt.EventRegisterDefaultImpl(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to create event listener")
	}
	go func() {
		for {
			err := libvirt.EventRunDefaultImpl()
			if err != nil {
				log.With(zap.Error(err)).DPanic("go func failed")
			}
		}
	}()
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.With(zap.Error(err)).DPanic("Failed to connect to libvirt")
	}
	defer conn.Close()

	lInstance := qemu.NewQemu(conn, log, imageLocation)

	if err := lInstance.InitializeBaseImagesAndNetwork(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to initialize Libvirt")
	}

	if err := lInstance.ExecuteCommands(); err != nil {
		log.With(zap.Error(err)).DPanic("Failed to run VMs")
	}
	log.Info("instaces terminated successfully")
}
