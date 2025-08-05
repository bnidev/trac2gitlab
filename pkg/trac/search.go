package trac

import (
	"fmt"
	"time"
)

type SearchFilter struct {
	Name        string
	Description string
}

func (c *Client) GetSearchFilters() ([]SearchFilter, error) {
	var resp [][]string

	err := c.rpc.Call("search.getSearchFilters", nil, &resp)
	if err != nil {
		return nil, err
	}

	var filters []SearchFilter
	for _, f := range resp {
		if len(f) == 2 {
			filters = append(filters, SearchFilter{
				Name:        f[0],
				Description: f[1],
			})
		}
	}

	return filters, nil
}

type SearchResult struct {
	Href    string
	Title   string
	Date    time.Time
	Author  string
	Excerpt string
}

func (c *Client) Search(query string) ([]SearchResult, error) {

	var resp [][]any

	err := c.rpc.Call("search.performSearch", []any{query}, &resp)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var results []SearchResult
	for _, r := range resp {
		if len(r) == 5 {
			if r[2] == "" {
				return nil, fmt.Errorf("empty date string in result: %v", r)
			}

			var href, title, author, excerpt string
			var date time.Time

			if r[0] != nil {
				href = r[0].(string)
			}
			if r[1] != nil {
				title = r[1].(string)
			}
			if r[2] != nil {
				date, _ = r[2].(time.Time)
			}
			if r[3] != nil {
				author = r[3].(string)
			}
			if r[4] != nil {
				excerpt = r[4].(string)
			}

			results = append(results, SearchResult{
				Href:    href,
				Title:   title,
				Date:    date,
				Author:  author,
				Excerpt: excerpt,
			})

		}
	}

	return results, nil
}
