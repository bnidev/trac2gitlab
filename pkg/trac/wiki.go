package trac


func (c *Client) GetWikiPageNames() ([]string, error) {
	var result []string
	err := c.rpc.Call("wiki.getAllPages", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
