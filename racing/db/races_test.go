package db

import (
	"database/sql"
	"git.neds.sh/matty/entain/racing/proto/racing"
	"os"
	"syreclabs.com/go/faker"
	"testing"
	"time"
)

const _racingTestsDB = "racing_tests.db"

// Setup inputs and expected outputs for TestApplyFilter test.
var applyFilterTests = []struct {
	name   string
	filter racing.ListRacesRequestFilter
	query  string
	args   []interface{}
}{
	{
		"empty_filter",
		racing.ListRacesRequestFilter{},
		"",
		[]interface{}{},
	},
	{
		"single_meeting_id",
		racing.ListRacesRequestFilter{MeetingIds: []int64{5}},
		" WHERE meeting_id IN (?)",
		[]interface{}{int64(5)},
	},
	{
		"multiple_meeting_id",
		racing.ListRacesRequestFilter{MeetingIds: []int64{5, 10}},
		" WHERE meeting_id IN (?,?)",
		[]interface{}{int64(5), int64(10)},
	},
	{
		"no_meeting_id_visible_only",
		racing.ListRacesRequestFilter{ShowVisibleOnly: true},
		" WHERE visible = 1",
		[]interface{}{},
	},
	{
		"single_meeting_id_visible_only",
		racing.ListRacesRequestFilter{MeetingIds: []int64{5}, ShowVisibleOnly: true},
		" WHERE meeting_id IN (?) AND visible = 1",
		[]interface{}{int64(5)},
	},
	{
		"multiple_meeting_id_visible_only",
		racing.ListRacesRequestFilter{MeetingIds: []int64{5, 10}, ShowVisibleOnly: true},
		" WHERE meeting_id IN (?,?) AND visible = 1",
		[]interface{}{int64(5), int64(10)},
	},
}

// Test the applyFilter method with various filters and validate correct query and arguments are returned.
func TestApplyFilter(t *testing.T) {
	var (
		r racesRepo
		q = getRaceQueries()[racesList]
	)

	// Execute the applyFilter method for each test input as a separate sub-test.
	for _, tc := range applyFilterTests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				query string
				args  []interface{}
			)

			query, args = r.applyFilter(q, &(tc.filter))

			// Validate the returned query matches the expected query.
			if query != q+tc.query {
				t.Errorf("Actual query %s does not match expected query %s", query, q+tc.query)
			}

			// Validate the number of returned args matches the number of expected args.
			if len(args) != len(tc.args) {
				t.Errorf("Actual args length %d does not match expected args length %d", len(args), len(tc.args))
			}

			// Validate the values of the returned args match the expected args.
			for i, arg := range tc.args {
				if arg != args[i] {
					t.Errorf("Actual args %s does not match expected args %s", args, tc.args)
				}
			}
		})
	}
}

// Setup inputs and expected outputs for TestApplyOrder test.
var applyOrderTests = []struct {
	name    string
	orderBy string
	query   string
}{
	{
		"empty_order_by_default",
		"",
		" ORDER BY advertised_start_time",
	},
	{
		"empty_order_by_fields_default",
		",",
		" ORDER BY advertised_start_time",
	},
	{
		"order_by_single_field",
		"meeting_id",
		" ORDER BY meeting_id",
	},
	{
		"order_by_single_field_desc",
		"meeting_id desc",
		" ORDER BY meeting_id desc",
	},
	{
		"order_by_multiple_fields",
		"meeting_id desc, advertised_start_time",
		" ORDER BY meeting_id desc, advertised_start_time",
	},
	{
		"remove_additional_spaces",
		"  meeting_id desc,  advertised_start_time  ",
		" ORDER BY meeting_id desc, advertised_start_time",
	},
	{
		"ignore_empty_fields",
		"meeting_id desc,, ,advertised_start_time",
		" ORDER BY meeting_id desc, advertised_start_time",
	},
}

// Test the applyOrder method and validate correct query is returned.
func TestApplyOrder(t *testing.T) {
	var (
		r racesRepo
		q = getRaceQueries()[racesList]
	)

	// Execute the applyFilter method for each test input as a separate sub-test.
	for _, tc := range applyOrderTests {
		t.Run(tc.name, func(t *testing.T) {
			query := r.applyOrder(q, tc.orderBy)

			// Validate the returned query matches the expected query.
			if query != q+tc.query {
				t.Errorf("Actual query %s does not match expected query %s", query, q+tc.query)
			}
		})
	}
}

// Setup inputs and expected outputs for TestList test.
var listTests = []struct {
	name    string
	filter  racing.ListRacesRequestFilter
	raceIds []int64
}{
	{
		"no_results",
		racing.ListRacesRequestFilter{MeetingIds: []int64{10}},
		nil, // no races
	},
	{
		"empty_filter",
		racing.ListRacesRequestFilter{},
		[]int64{1, 2, 3, 4}, // all races
	},
	{
		"single_meeting_id",
		racing.ListRacesRequestFilter{MeetingIds: []int64{1}},
		[]int64{1, 3}, // races 1 and 3
	},
	{
		"multiple_meeting_id",
		racing.ListRacesRequestFilter{MeetingIds: []int64{1, 5}},
		[]int64{1, 2, 3}, // race 1, 2 and 3
	},
	{
		"no_meeting_id_visible_only",
		racing.ListRacesRequestFilter{ShowVisibleOnly: true},
		[]int64{1, 2}, // race 1 and 2
	},
	{
		"single_meeting_id_visible_only",
		racing.ListRacesRequestFilter{MeetingIds: []int64{1}, ShowVisibleOnly: true},
		[]int64{1}, // race 1 only
	},
	{
		"multiple_meeting_id_visible_only",
		racing.ListRacesRequestFilter{MeetingIds: []int64{5, 6}, ShowVisibleOnly: true},
		[]int64{2}, // race 2 only
	},
}

// Tests the List method applying various filters return the correct collection of races ordered by race ID.
func TestList(t *testing.T) {
	// Setup test data in random order
	var testData = []struct {
		id        int64
		meetingID int64
		visible   int8
	}{
		{1, 1, 1}, // Visible
		{2, 5, 1}, // Visible
		{4, 6, 0}, // Not visible
		{3, 1, 0}, // Not visible
	}

	// Setup test database.
	// If the database file already exists, the file will be truncated.
	file, err := os.Create(_racingTestsDB)
	if err != nil {
		t.Fatalf("Could not create test database. %s", err)
	}

	// Tear down test database on test completion.
	defer file.Close()
	defer os.Remove(_racingTestsDB)

	racingTestDB, err := sql.Open("sqlite3", _racingTestsDB)
	if err != nil {
		t.Fatalf("Could not open test database. %s", err)
	}

	statement, err := racingTestDB.Prepare(`CREATE TABLE IF NOT EXISTS races (id INTEGER PRIMARY KEY, meeting_id INTEGER, name TEXT, number INTEGER, visible INTEGER, advertised_start_time DATETIME)`)
	if err == nil {
		_, err = statement.Exec()
	}

	for _, testRow := range testData {
		statement, err = racingTestDB.Prepare(`INSERT OR IGNORE INTO races(id, meeting_id, name, number, visible, advertised_start_time) VALUES (?,?,?,?,?,?)`)
		if err == nil {
			_, err = statement.Exec(
				testRow.id,
				testRow.meetingID,
				faker.Team().Name(),
				faker.Number().Between(1, 12),
				testRow.visible,
				faker.Time().Between(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2)).Format(time.RFC3339),
			)
		}
	}

	if err != nil {
		t.Fatalf("Could not setup test database. %s", err)
	}

	racesRepo := NewRacesRepo(racingTestDB)

	// Run each test input as a separate sub-test.
	for _, tc := range listTests {
		t.Run(tc.name, func(t *testing.T) {
			// Execute the List method using the input filter and ordering by race ID
			races, err := racesRepo.List(&tc.filter, "id")
			if err != nil {
				t.Errorf("Expected race results but an error occurred. %s", err)
			}

			// Validate the actual number of races returned in the response.
			// matches the expected number of races
			if len(races) != len(tc.raceIds) {
				t.Errorf("Actual race count %d does not match expected race count %d", len(races), len(tc.raceIds))
			}

			for i, race := range races {
				// Validate that all returned races are visible if the ShowVisibleOnly filter is applied.
				if tc.filter.ShowVisibleOnly {
					if !race.Visible {
						t.Error("Returned race is not visible but expected only visible races.")
					}
				}

				// Validate the race ID returned matches the expected race ID based on the expected order
				if race.Id != tc.raceIds[i] {
					t.Errorf("Actual race ID %d does not match expected race ID %d", race.Id, tc.raceIds[i])
				}
			}
		})
	}
}
