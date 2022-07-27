package db

import (
	"database/sql"
	"git.neds.sh/matty/entain/sports/proto/sports"
	"os"
	"syreclabs.com/go/faker"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const _sportTestsDB = "sports_tests.db"

// Setup inputs and expected outputs for TestApplyFilter test.
var applyFilterTests = []struct {
	name   string
	filter *sports.ListEventsRequestFilter
	query  string
	args   []interface{}
}{
	{
		"empty_filter",
		&sports.ListEventsRequestFilter{},
		"",
		[]interface{}{},
	},
	{
		"single_sport_id",
		&sports.ListEventsRequestFilter{SportIds: []int64{5}},
		" WHERE sport_id IN (?)",
		[]interface{}{int64(5)},
	},
	{
		"multiple_sport_ids",
		&sports.ListEventsRequestFilter{SportIds: []int64{5, 10}},
		" WHERE sport_id IN (?,?)",
		[]interface{}{int64(5), int64(10)},
	},
}

// Test the applyFilter method with various filters and validate correct query and arguments are returned.
func TestApplyFilter(t *testing.T) {
	var (
		r sportsRepo
		q = getEventQueries()[eventsList]
	)

	// Execute the applyFilter method for each test input as a separate sub-test.
	for _, tc := range applyFilterTests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				query string
				args  []interface{}
			)

			query, args = r.applyFilter(q, tc.filter)

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
		"sport_id",
		" ORDER BY sport_id",
	},
	{
		"order_by_single_field_desc",
		"sport_id desc",
		" ORDER BY sport_id desc",
	},
	{
		"order_by_multiple_fields",
		"sport_id desc, advertised_start_time",
		" ORDER BY sport_id desc, advertised_start_time",
	},
	{
		"remove_additional_spaces",
		"  sport_id desc,  advertised_start_time  ",
		" ORDER BY sport_id desc, advertised_start_time",
	},
	{
		"ignore_empty_fields",
		"sport_id desc,, ,advertised_start_time",
		" ORDER BY sport_id desc, advertised_start_time",
	},
}

// Test the applyOrder method and validate correct query is returned.
func TestApplyOrder(t *testing.T) {
	var (
		r sportsRepo
		q = getEventQueries()[eventsList]
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
	name     string
	filter   *sports.ListEventsRequestFilter
	eventIds []int64
}{
	{
		"no_results",
		&sports.ListEventsRequestFilter{SportIds: []int64{10}},
		nil, // no events
	},
	{
		"empty_filter",
		&sports.ListEventsRequestFilter{},
		[]int64{1, 2, 3, 4}, // all events
	},
	{
		"single_sport_id",
		&sports.ListEventsRequestFilter{SportIds: []int64{1}},
		[]int64{1, 3}, // events 1 and 3
	},
	{
		"multiple_sport_ids",
		&sports.ListEventsRequestFilter{SportIds: []int64{1, 5}},
		[]int64{1, 2, 3}, // event 1, 2 and 3
	},
}

// Tests the List method applying various filters return the correct collection of events ordered by event ID.
func TestList(t *testing.T) {
	// Setup and teardown test database
	sportsTestDB := setupDb(t)

	// Setup test data in random order
	var testData = []struct {
		id      int64
		sportID int64
	}{
		{1, 1},
		{2, 5},
		{4, 6},
		{3, 1},
	}

	for _, testRow := range testData {
		statement, err := sportsTestDB.Prepare(`INSERT OR IGNORE INTO events(id, sport_id, name, advertised_start_time, status) VALUES (?,?,?,?,?)`)
		if err != nil {
			t.Fatalf("Could not setup test data. %s", err)
		}

		_, err = statement.Exec(
			testRow.id,
			testRow.sportID,
			faker.Team().Name(),
			faker.Time().Between(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2)).Format(time.RFC3339),
			faker.Number().Between(0, 3),
		)

		if err != nil {
			t.Fatalf("Could not setup test data. %s", err)
		}
	}

	sportsRepo := NewSportsRepo(sportsTestDB)

	// Run each test input as a separate sub-test.
	for _, tc := range listTests {
		t.Run(tc.name, func(t *testing.T) {
			// Execute the List method using the input filter and ordering by event ID
			events, err := sportsRepo.List(tc.filter, "id")
			if err != nil {
				t.Errorf("Expected event results but an error occurred. %s", err)
			}

			// Validate the actual number of events returned in the response
			// matches the expected number of events.
			if len(events) != len(tc.eventIds) {
				t.Errorf("Actual event count %d does not match expected event count %d", len(events), len(tc.eventIds))
			}

			for i, event := range events {
				// Validate the event ID returned matches the expected event ID based on the expected order
				if event.Id != tc.eventIds[i] {
					t.Errorf("Actual event ID %d does not match expected event ID %d", event.Id, tc.eventIds[i])
				}
			}
		})
	}
}

// Tests the Get method to validate it returns a event for the given ID.
func TestGet(t *testing.T) {
	// Setup and teardown test database
	sportsTestDB := setupDb(t)

	// Setup test data
	statement, err := sportsTestDB.Prepare(`INSERT OR IGNORE INTO events(id, sport_id, name, advertised_start_time, status) VALUES (?,?,?,?,?)`)
	if err != nil {
		t.Fatalf("Could not setup test data. %s", err)
	}

	_, err = statement.Exec(
		1,
		faker.Number().Between(1, 10),
		faker.Team().Name(),
		faker.Time().Between(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2)).Format(time.RFC3339),
		faker.Number().Between(0, 3),
	)

	if err != nil {
		t.Fatalf("Could not setup test data. %s", err)
	}

	sportsRepo := NewSportsRepo(sportsTestDB)

	// Execute the Get method and validate event with ID 1 is returned.
	event, err := sportsRepo.Get(1)
	if err != nil || event.Id != 1 {
		t.Errorf("Expected event with ID 1 to be returned. %s", err)
	}
}

// Tests the Get method to validate it returns a NotFound error if no event is found.
func TestGetNotFound(t *testing.T) {
	// Setup and teardown test database
	sportsTestDB := setupDb(t)

	sportsRepo := NewSportsRepo(sportsTestDB)

	// Execute the Get method and validate a NotFound error occurs.
	event, err := sportsRepo.Get(1)
	if event != nil || status.Code(err) != codes.NotFound {
		t.Errorf("Expected a NotFound error. %s", err)
	}
}

// setupDb prepares a test database and registers a cleanup task to delete the test database.
func setupDb(t *testing.T) *sql.DB {
	var (
		dbFile *os.File
		err    error
	)

	// Tear down test database on completion
	t.Cleanup(func() {
		if dbFile != nil {
			dbFile.Close()
		}

		os.Remove(_sportTestsDB)
	})

	// Setup test database.
	// If the database file already exists, the file will be truncated.
	dbFile, err = os.Create(_sportTestsDB)
	if err != nil {
		t.Fatalf("Could not create test database. %s", err)
	}

	sportsTestDB, err := sql.Open("sqlite3", _sportTestsDB)
	if err != nil {
		t.Fatalf("Could not open test database. %s", err)
	}

	statement, err := sportsTestDB.Prepare(`CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY, sport_id INTEGER, name TEXT, advertised_start_time DATETIME, status INTEGER)`)
	if err == nil {
		_, err = statement.Exec()
	}

	return sportsTestDB
}
