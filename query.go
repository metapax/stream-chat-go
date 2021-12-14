package stream_chat // nolint: golint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type QueryOption struct {
	// https://getstream.io/chat/docs/#query_syntax
	Filter map[string]interface{} `json:"filter_conditions,omitempty"`
	Sort   []*SortOption          `json:"sort,omitempty"`

	UserID       string `json:"user_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`  // pagination option: limit number of results
	Offset       int    `json:"offset,omitempty"` // pagination option: offset to return items from
	MessageLimit *int   `json:"message_limit,omitempty"`
	MemberLimit  *int   `json:"member_limit,omitempty"`
}

type SortOption struct {
	Field     string `json:"field"`     // field name to sort by,from json tags(in camel case), for example created_at
	Direction int    `json:"direction"` // [-1, 1]
}

type queryRequest struct {
	Watch    bool `json:"watch"`
	State    bool `json:"state"`
	Presence bool `json:"presence"`

	UserID       string `json:"user_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	MemberLimit  *int   `json:"member_limit,omitempty"`
	MessageLimit *int   `json:"message_limit,omitempty"`

	FilterConditions map[string]interface{} `json:"filter_conditions,omitempty"`
	Sort             []*SortOption          `json:"sort,omitempty"`
}

type queryUsersResponse struct {
	Users []*User `json:"users"`
}

// QueryUsers returns list of users that match QueryOption.
// If any number of SortOption are set, result will be sorted by field and direction in the order of sort options.
func (c *Client) QueryUsers(ctx context.Context, q *QueryOption, sorters ...*SortOption) ([]*User, error) {
	qp := queryRequest{
		FilterConditions: q.Filter,
		Limit:            q.Limit,
		Offset:           q.Offset,
		Sort:             sorters,
	}

	data, err := json.Marshal(&qp)
	if err != nil {
		return nil, err
	}

	values := make(url.Values)
	values.Set("payload", string(data))

	var resp queryUsersResponse
	err = c.makeRequest(ctx, http.MethodGet, "users", values, nil, &resp)

	return resp.Users, err
}

type queryChannelResponse struct {
	Channels []queryChannelResponseData `json:"channels"`
}

type queryChannelResponseData struct {
	Channel  *Channel         `json:"channel"`
	Messages []*Message       `json:"messages"`
	Read     []*ChannelRead   `json:"read"`
	Members  []*ChannelMember `json:"members"`
}

// QueryChannels returns list of channels with members and messages, that match QueryOption.
// If any number of SortOption are set, result will be sorted by field and direction in oder of sort options.
func (c *Client) QueryChannels(ctx context.Context, q *QueryOption, sort ...*SortOption) ([]*Channel, error) {
	qp := queryRequest{
		State:            true,
		FilterConditions: q.Filter,
		Sort:             sort,
		UserID:           q.UserID,
		Limit:            q.Limit,
		Offset:           q.Offset,
		MemberLimit:      q.MemberLimit,
		MessageLimit:     q.MessageLimit,
	}

	var resp queryChannelResponse
	if err := c.makeRequest(ctx, http.MethodPost, "channels", nil, qp, &resp); err != nil {
		return nil, err
	}

	result := make([]*Channel, len(resp.Channels))
	for i, data := range resp.Channels {
		result[i] = data.Channel
		result[i].Members = data.Members
		result[i].Messages = data.Messages
		result[i].Read = data.Read
		result[i].client = c
	}

	return result, nil
}

type SearchRequest struct {
	// Required
	Query          string                 `json:"query"`
	Filters        map[string]interface{} `json:"filter_conditions"`
	MessageFilters map[string]interface{} `json:"message_filter_conditions"`

	// Pagination, optional
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Next   string `json:"next,omitempty"`

	// Sort, optional
	Sort []SortOption `json:"sort,omitempty"`
}

type SearchResponse struct {
	Results  []SearchMessageResponse `json:"results"`
	Next     string                  `json:"next,omitempty"`
	Previous string                  `json:"previous,omitempty"`
}

type SearchMessageResponse struct {
	Message *Message `json:"message"`
}

// Search returns channels matching for given keyword.
func (c *Client) Search(ctx context.Context, request SearchRequest) ([]*Message, error) {
	result, err := c.SearchWithFullResponse(ctx, request)
	if err != nil {
		return nil, err
	}
	messages := make([]*Message, 0, len(result.Results))
	for _, res := range result.Results {
		messages = append(messages, res.Message)
	}

	return messages, nil
}

// SearchWithFullResponse performs a search and returns the full results.
func (c *Client) SearchWithFullResponse(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	if request.Offset != 0 {
		if len(request.Sort) > 0 || request.Next != "" {
			return nil, errors.New("cannot use Offset with Next or Sort parameters")
		}
	}
	if request.Query != "" && len(request.MessageFilters) != 0 {
		return nil, errors.New("can only specify Query or MessageFilters, not both")
	}
	var buf strings.Builder

	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set("payload", buf.String())

	var result SearchResponse
	if err := c.makeRequest(ctx, http.MethodGet, "search", values, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type queryMessageFlagsResponse struct {
	Flags []*MessageFlag `json:"flags"`
}

// QueryMessageFlags returns list of message flags that match QueryOption.
func (c *Client) QueryMessageFlags(ctx context.Context, q *QueryOption) ([]*MessageFlag, error) {
	qp := queryRequest{
		FilterConditions: q.Filter,
		Limit:            q.Limit,
		Offset:           q.Offset,
	}

	data, err := json.Marshal(&qp)
	if err != nil {
		return nil, err
	}

	values := make(url.Values)
	values.Set("payload", string(data))

	var resp queryMessageFlagsResponse
	err = c.makeRequest(ctx, http.MethodGet, "moderation/flags/message", values, nil, &resp)

	return resp.Flags, err
}
