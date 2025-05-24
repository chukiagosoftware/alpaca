package models

type ExternalLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
	Description string `json:"description"`
	Category string `json:"category"`
	Tags []string `json:"tags"`
	Rating int64 `json:"rating"`
	HotelId int64 `json:"hotelId"`
	CreatedAt string `json:"createdAt"`

}

type Hotel struct {
	Id            int64  `json:"id"`
	CreatedAt     string `json:"createdAt"`
	Icon          string `json:"icon"`
	Location      string `json:"location"`
	Name          string `json:"name"`
	Description  string  `json:"description"`
	Rating		 int64 `json:"rating"`
	Comments	 []string `json:"comments"`
	Warnings	 []string `json:"comments"`
	Bonus		 []string `json:"comments"`
	ExternalLinks []ExternalLink
	Tags		 []string `json:"tags"`
}
