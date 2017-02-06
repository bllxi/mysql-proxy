package proxy

import "log"

func (c *Client) handleUseDB(db string) error {
	log.Printf("use %s\n", db)
	return c.writeOK(nil)
}
