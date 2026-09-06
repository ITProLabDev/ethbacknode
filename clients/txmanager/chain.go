package txmanager

// ChainNonces is the one question the allocator asks the chain: how many
// transactions has an address already sent, counting ones still in the
// node's pool. *ethclient.Client already answers this (PendingNonceAt);
// declared here, rather than imported from there, so a test can answer it in
// three lines and so this package states what it actually needs.
type ChainNonces interface {
	PendingNonceAt(address string) (int64, error)
}
