package postgres

// Rest of the file remains unchanged

func (r *repository) GetURLMetrics(ctx context.Context, url string, since time.Time) (*content.URLMetrics, error) {
	const op = "ContentRepository.GetURLMetrics"

	var metrics content.URLMetrics
	metrics.URL = url
	metrics.ErrorRates = make(map[string]float64)

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		// Get load and error counts
		err := tx.QueryRowContext(ctx, `
			SELECT 
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED'),
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR')
			FROM content_events 
			WHERE url = $1 AND timestamp >= $2
		`, url, since).Scan(&metrics.LoadCount, &metrics.ErrorCount)
		if err != nil {
			return err
		}

		// Get last seen timestamp
		err = tx.QueryRowContext(ctx, `
			SELECT EXTRACT(EPOCH FROM MAX(timestamp))::bigint
			FROM content_events 
			WHERE url = $1
		`, url).Scan(&metrics.LastSeen)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get average timing metrics
		err = tx.QueryRowContext(ctx, `
			SELECT 
				COALESCE(AVG((metrics->>'loadTime')::numeric), 0),
				COALESCE(AVG((metrics->>'renderTime')::numeric), 0)
			FROM content_events 
			WHERE url = $1 
				AND timestamp >= $2
				AND type = 'CONTENT_LOADED'
				AND metrics IS NOT NULL
				AND metrics->>'loadTime' IS NOT NULL
				AND metrics->>'renderTime' IS NOT NULL
		`, url, since).Scan(&metrics.AvgLoadTime, &metrics.AvgRenderTime)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get error rates by error code
		rows, err := tx.QueryContext(ctx, `
			WITH error_counts AS (
				SELECT 
					error->>'code' as error_code,
					COUNT(*) as error_count
				FROM content_events 
				WHERE url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_ERROR'
					AND error IS NOT NULL
					AND error->>'code' IS NOT NULL
				GROUP BY error->>'code'
			),
			total_events AS (
				SELECT COUNT(*)::float AS total
				FROM content_events
				WHERE url = $1 AND timestamp >= $2
			)
			SELECT 
				error_code,
				error_count::float / NULLIF(total, 0) as error_rate
			FROM error_counts, total_events
			WHERE error_code IS NOT NULL
		`, url, since)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var code string
			var rate float64
			if err := rows.Scan(&code, &rate); err != nil {
				return err
			}
			metrics.ErrorRates[code] = rate
		}

		return rows.Err()
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return &metrics, nil
}
