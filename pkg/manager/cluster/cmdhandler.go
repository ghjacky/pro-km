package cluster

func (m *manager) addCmdHandlers() {
	go m.pullData()
}
