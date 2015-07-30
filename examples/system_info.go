package main

import (
	"fmt"
	"github.com/abimaelmartell/gosigar"
)

func main() {
	concreteSigar := sigar.ConcreteSigar{}
	systemInfo, _ := concreteSigar.GetSystemInfo()

	fmt.Println("Name:", systemInfo.Name)
	fmt.Println("Version:", systemInfo.Version)
	fmt.Println("Arch:", systemInfo.Arch)
	fmt.Println("Machine:", systemInfo.Machine)
	fmt.Println("Description:", systemInfo.Description)
	fmt.Println("PatchLevel:", systemInfo.PatchLevel)
	fmt.Println("Vendor:", systemInfo.Vendor)
	fmt.Println("VendorVersion:", systemInfo.VendorVersion)
	fmt.Println("VendorName:", systemInfo.VendorName)
	fmt.Println("VendorCodeName:", systemInfo.VendorCodeName)
}
