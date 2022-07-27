## Sports Service

### ListEvents

`POST:/v1/list-events`

This endpoint allows users to retrieve a list of sports events and optionally provide filters and a sort order.

It accepts a JSON request body with optional `filter` and `order_by` properties as shown in the example below.

```
{
    "filter": {
        "sport_ids": [1, 5]
    },
    "order_by": "sport_id desc"
}
```

The events listed can be filtered by one or more Sport ID values by specifying the `sport_ids` field in the `filter` and providing an array of integers.

To order events by a particular field or set of fields, specify the `order_by` field in `filter`. This accepts a comma-delimited list of field names. For descending order, apply a ` desc` suffix to the field name. If no order is specified, the results will be ordered by the `advertised_start_time` by default.

`GET:/v1/events/:id`

This endpoint allows users to retrieve a single event with the ID specified in the endpoint URL. A 404 `NotFound` error will be returned if no event exists with the provided ID.