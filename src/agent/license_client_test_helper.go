package agent

// SetActivationForTest sets the activation data and SN for testing purposes.
// This method should only be used in tests.
func (c *LicenseClient) SetActivationForTest(data *ActivationData, sn string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.sn = sn
}
