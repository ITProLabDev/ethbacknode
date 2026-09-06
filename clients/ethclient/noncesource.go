package ethclient

// NonceSource supplies the nonce to sign a from address's next transaction
// with, and learns what became of one it allocated. When no NonceSource is
// wired (SetNonceSource never called), the client falls back to asking the
// node directly via PendingNonceAt on every send -- correct for one
// transaction at a time and wrong for two prepared back to back, which is
// what clients/txmanager exists to fix.
type NonceSource interface {
	// Allocate returns the nonce to sign from's next transaction with.
	Allocate(from string) (int64, error)
	// Sent records that a transaction under nonce was broadcast successfully.
	Sent(from, txHash string, nonce int64)
	// Release takes back a nonce that was allocated but never reached the
	// node (signing or broadcasting failed), so it can be reused.
	Release(from string, nonce int64)
}

// SetNonceSource wires the allocator this client signs nonces from. The
// client and the allocator are typically mutually dependent at construction
// (the allocator seeds from the client), so this is a setter completed after
// both exist, rather than a NewClient option.
func (c *Client) SetNonceSource(s NonceSource) { c.nonceSource = s }

// allocateNonce returns the nonce to sign from's next transaction with,
// through the wired NonceSource, or directly from the node when none is
// wired.
func (c *Client) allocateNonce(from string) (int64, error) {
	if c.nonceSource != nil {
		return c.nonceSource.Allocate(from)
	}
	return c.PendingNonceAt(from)
}

func (c *Client) releaseNonce(from string, nonce int64) {
	if c.nonceSource != nil {
		c.nonceSource.Release(from, nonce)
	}
}

func (c *Client) recordSent(from, txHash string, nonce int64) {
	if c.nonceSource != nil {
		c.nonceSource.Sent(from, txHash, nonce)
	}
}

// withAllocatedNonce allocates a nonce for from and runs fn with it. fn's
// nonce is released when it returns an error -- the chain never saw it, so
// the next transaction from this address must not be signed above a gap --
// and recorded as sent when it succeeds.
func (c *Client) withAllocatedNonce(from string, fn func(nonce int64) (txHash string, err error)) (string, error) {
	nonce, err := c.allocateNonce(from)
	if err != nil {
		return "", err
	}
	txHash, err := fn(nonce)
	if err != nil {
		c.releaseNonce(from, nonce)
		return "", err
	}
	c.recordSent(from, txHash, nonce)
	return txHash, nil
}
