type VDIType int

const (
	_ VDIType = iota
	Disk
	CD
	Floppy
)

// Expose a VDI using the Transfer VM
// (Legacy VHD export)

type TransferRecord struct {
	UrlFull string `xml:"url_full,attr"`
}

func (self *VDI) Expose(format string) (url string, err error) {

	hosts, err := self.Client.GetHosts()

	if err != nil {
		err = errors.New(fmt.Sprintf("Could not retrieve hosts in the pool: %s", err.Error()))
		return "", err
	}
	host := hosts[0]

	disk_uuid, err := self.GetUuid()

	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get VDI uuid for %s: %s", self.Ref, err.Error()))
		return "", err
	}

	args := make(map[string]string)
	args["transfer_mode"] = "http"
	args["vdi_uuid"] = disk_uuid
	args["expose_vhd"] = "true"
	args["network_uuid"] = "management"
	args["timeout_minutes"] = "5"

	handle, err := host.CallPlugin("transfer", "expose", args)

	if err != nil {
		err = errors.New(fmt.Sprintf("Error whilst exposing VDI %s: %s", disk_uuid, err.Error()))
		return "", err
	}

	args = make(map[string]string)
	args["record_handle"] = handle
	record_xml, err := host.CallPlugin("transfer", "get_record", args)

	if err != nil {
		err = errors.New(fmt.Sprintf("Unable to retrieve transfer record for VDI %s: %s", disk_uuid, err.Error()))
		return "", err
	}

	var record TransferRecord
	xml.Unmarshal([]byte(record_xml), &record)

	if record.UrlFull == "" {
		return "", errors.New(fmt.Sprintf("Error: did not parse XML properly: '%s'", record_xml))
	}

	// Handles either raw or VHD formats

	switch format {
	case "vhd":
		url = fmt.Sprintf("%s.vhd", record.UrlFull)

	case "raw":
		url = record.UrlFull
	}

	return
}

// Unexpose a VDI if exposed with a Transfer VM.

func (self *VDI) Unexpose() (err error) {

	disk_uuid, err := self.GetUuid()

	if err != nil {
		return err
	}

	hosts, err := self.Client.GetHosts()

	if err != nil {
		err = errors.New(fmt.Sprintf("Could not retrieve hosts in the pool: %s", err.Error()))
		return err
	}

	host := hosts[0]

	args := make(map[string]string)
	args["vdi_uuid"] = disk_uuid

	result, err := host.CallPlugin("transfer", "unexpose", args)

	if err != nil {
		return err
	}

	log.Println(fmt.Sprintf("Unexpose result: %s", result))

	return nil
}
