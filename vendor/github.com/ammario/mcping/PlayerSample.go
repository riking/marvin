package mcping

//PlayerSample contains a player response in the sample section of a ping response.
type PlayerSample struct {
	UUID string `json:"uuid"` //e.g "d8a973a5-4c0f-4af6-b1ea-0a76cd210cc5"
	Name string `json:"name"` //e.g "Ammar"
}
