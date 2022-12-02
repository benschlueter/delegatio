package main

import (
	"fmt"
	"log"

	"libvirt.org/go/libvirt"
)

func main() {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	domain, err := conn.LookupDomainByName("delegatio-vm")
	if err != nil {
		log.Panic(err)
	}
	output, err := domain.QemuAgentCommand("{\"execute\":\"guest-info\"}", libvirt.DOMAIN_QEMU_AGENT_COMMAND_BLOCK, 0)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(output)
}
