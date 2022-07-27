package db

const (
	racesList = "list"
)

func getRaceQueries() map[string]string {
	return map[string]string{
		// status calculated field returns:
		//   - 0 (OPEN) if advertised start time is >= the current date/time
		//   - 1 (CLOSED) if advertised start is < the current date/time
		racesList: `
			SELECT 
				id, 
				meeting_id, 
				name, 
				number, 
				visible, 
				advertised_start_time, 
				CASE WHEN datetime(advertised_start_time, 'localtime') >= datetime('now','localtime') THEN 0 ELSE 1 END status 
			FROM races
		`,
	}
}
