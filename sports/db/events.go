package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
	_ "github.com/mattn/go-sqlite3"

	"git.neds.sh/matty/entain/sports/proto/sports"
)

// SportsRepo provides repository access to sport events.
type SportsRepo interface {
	// Init will initialise our sports repository.
	Init() error

	// List will return a list of events.
	List(filter *sports.ListEventsRequestFilter, orderBy string) ([]*sports.Event, error)

	// Get will return a single event by event ID.
	Get(id int64) (*sports.Event, error)
}

type sportsRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewSportsRepo creates a new sports repository.
func NewSportsRepo(db *sql.DB) SportsRepo {
	return &sportsRepo{db: db}
}

// Init prepares the sports repository dummy data.
func (r *sportsRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy sports events.
		err = r.seed()
	})

	return err
}

// List returns a collection of sports events that match the provided filter
// or all events if no filter is provided. The results will be ordered by
// advertised start time by default or the order as specified in the request.
func (r *sportsRepo) List(filter *sports.ListEventsRequestFilter, orderBy string) ([]*sports.Event, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getEventQueries()[eventsList]

	query, args = r.applyFilter(query, filter)
	query = r.applyOrder(query, orderBy)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanEvents(rows)
}

func (r *sportsRepo) Get(id int64) (*sports.Event, error) {
	var (
		err   error
		query string
		args  []interface{}
	)

	query = getEventQueries()[eventsList]

	query += " WHERE id = ?"
	args = append(args, id)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	// Use the existing scanEvents method which will return an array
	// with one element if the event is found or 0 elements if not found.
	events, err := r.scanEvents(rows)

	if len(events) == 0 {
		// Event was not found
		err = status.Error(codes.NotFound, "Event was not found")
		return nil, err
	}

	// Event was found
	return events[0], err
}

// applyFilter returns the formulated query and query parameter values
// based on the provided filter values.
func (r *sportsRepo) applyFilter(query string, filter *sports.ListEventsRequestFilter) (string, []interface{}) {
	var (
		clauses []string
		args    []interface{}
	)

	if filter == nil {
		return query, args
	}

	if len(filter.SportIds) > 0 {
		clauses = append(clauses, "sport_id IN ("+strings.Repeat("?,", len(filter.SportIds)-1)+"?)")

		for _, sportID := range filter.SportIds {
			args = append(args, sportID)
		}
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

// applyOrder returns the query with an order by clause appended based on the order specified in the request.
func (r *sportsRepo) applyOrder(query string, orderBy string) string {
	// If no order is specified, order by advertised start time.
	if strings.TrimSpace(orderBy) == "" {
		query += " ORDER BY advertised_start_time"
		return query
	}

	// Split the order by string by comma into an array
	fields := strings.Split(orderBy, ",")

	var validFields []string

	// Trim any whitespace around each order by field
	for _, value := range fields {
		value = strings.TrimSpace(strings.Replace(value, "  ", " ", -1))
		if value != "" {
			validFields = append(validFields, value)
		}
	}

	if len(validFields) == 0 {
		query += " ORDER BY advertised_start_time"
		return query
	}

	query += " ORDER BY " + strings.Join(validFields, ", ")
	return query
}

func (m *sportsRepo) scanEvents(
	rows *sql.Rows,
) ([]*sports.Event, error) {
	var events []*sports.Event

	for rows.Next() {
		var event sports.Event
		var advertisedStart time.Time

		if err := rows.Scan(&event.Id, &event.SportId, &event.Name, &advertisedStart, &event.Status); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return nil, err
		}

		ts, err := ptypes.TimestampProto(advertisedStart)
		if err != nil {
			return nil, err
		}

		event.AdvertisedStartTime = ts

		events = append(events, &event)
	}

	return events, nil
}
