## Racing Service

### ListRaces

`POST:/v1/list-races`

This endpoint allows users to retrieve a list of races and optionally provide filters and a sort order.

It accepts a JSON request body with optional `filter` and `order_by` properties as shown in the example below.

```
{
    "filter": {
        "meeting_ids": [1, 5],
        "show_visible_only": true
    },
    "order_by": "meeting_id desc"
}
```

The races listed can be filtered by one or more Meeting ID values by specifying the `meeting_ids` field in the `filter` and providing an array of integers.

Races can also be limited to visible races only by using the `show_visible_only` field in the `filter` with a value of `true`.

To order races by a particular field or set of fields, specify the `order_by` field in `filter`. This accepts a comma-delimited list of field names. For descending order, apply a ` desc` suffix to the field name. If no order is specified, the results will be ordered by the `advertised_start_time` by default.

`GET:/v1/races/:id`

This endpoint allows users to retrieve a single race with the ID specified in the endpoint URL. A 404 `NotFound` error will be returned if no race exists with the provided ID.