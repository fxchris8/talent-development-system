package web

type SeamanLookupResponse struct {
	Name       string `json:"name" gorm:"column:name"`
	SeamanCode string `json:"seamanCode" gorm:"column:seaman_code"`
	// AlreadyAdded is true when a report for this seaman exists (such seamen cannot be added again).
	AlreadyAdded bool `json:"alreadyAdded" gorm:"column:already_added"`
}
