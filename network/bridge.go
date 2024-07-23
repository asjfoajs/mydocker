package network

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	//TODO implement me
	panic("implement me")
}

func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (d *BridgeNetworkDriver) Disconnect(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (d *BridgeNetworkDriver) Delete(network *Network) error {
	//TODO implement me
	panic("implement me")
}
