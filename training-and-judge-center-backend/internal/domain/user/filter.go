package user

type UserFilter struct {
	Roles       []Role
	Status      *Status
	Country     string
	City        string
	Institution string
	SearchField string // "name"|"nickname"|"email"|"institution"|"all"
	SearchTerm  string
	Sort        string // "createdAt"|"name"|"nickname"|"email"|"deactivatedAt"
	Order       string // "asc"|"desc"
	Page        int
	Limit       int
}

var ValidSortFields = map[string]bool{
	"createdAt":     true,
	"name":          true,
	"nickname":      true,
	"email":         true,
	"deactivatedAt": true,
}

var ValidSearchFields = map[string]bool{
	"name":        true,
	"nickname":    true,
	"email":       true,
	"institution": true,
	"all":         true,
}
