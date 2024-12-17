package api

import "fmt"

type Keys struct {
	SecretAPIKey string `json:"secretapikey"`
	APIKey       string `json:"apikey"`
}

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type PingRequest struct {
	Keys
}

type PingResponse struct {
	Status
	YourIP string `json:"yourIp"`
}

type Record struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio"`
	Notes   string `json:"notes"`
}

type RecordsRequest struct {
	Keys
}

type RecordsResponse struct {
	Status
	Records []*Record `json:"records"`
}

type CreateResponse struct {
	Status
	ID string `json:"id"`
}

type UpdateRequest struct {
	Keys

	// The subdomain for the record being created, not including the domain itself.
	// Leave blank to create a record on the root domain.
	// Use * to create a wildcard record.
	Name string `json:"name"`

	// The type of record being created.
	// Valid types are: A, MX, CNAME, ALIAS, TXT, NS, AAAA, SRV, TLSA, CAA, HTTPS, SVCB
	Type string `json:"type"`

	// The answer content for the record.
	// Please see the Porkbun documentation for proper formatting of each record type.
	Content string `json:"content"`

	// The time to live in seconds for the record.
	// The minimum and the default is 600 seconds.
	TTL string `json:"ttl"`

	// (optional) The priority of the record for those that support it.
	Prio string `json:"prio"`
}

type EditResponse struct {
	Status
}

func (r *Record) String() string {
	return fmt.Sprintf("%s %s %s %s %s (%s)", r.Name, r.Type, r.Content, r.TTL, r.Prio, r.ID)
}
