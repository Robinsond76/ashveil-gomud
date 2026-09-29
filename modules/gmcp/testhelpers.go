package gmcp

// AcceptGMCPForTest marks a connection as having accepted GMCP, as a web
// client's connection is from the start, for tests in other packages.
func AcceptGMCPForTest(connectionId uint64) {
	settings, _ := gmcpModule.cache.Get(connectionId)
	settings.GMCPAccepted = true
	gmcpModule.cache.Add(connectionId, settings)
}

// RefuseGMCPForTest marks a connection as one that hasn't accepted GMCP (a
// plain telnet client), for tests in other packages.
func RefuseGMCPForTest(connectionId uint64) {
	settings, _ := gmcpModule.cache.Get(connectionId)
	settings.GMCPAccepted = false
	gmcpModule.cache.Add(connectionId, settings)
}
