func (self *Network) IsHostInternalManagementNetwork() (isHostInternalManagementNetwork bool, err error) {
	other_config, err := self.GetOtherConfig()
	if err != nil {
		return false, nil
	}
	value, ok := other_config["is_host_internal_management_network"]
	isHostInternalManagementNetwork = ok && value == "true"
	return isHostInternalManagementNetwork, nil
}
