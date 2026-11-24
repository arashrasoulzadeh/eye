package model

// CompactedEventBucket stores counts for raw events removed by compaction.
// Kept separate from live buckets so rebuilding from raw events is safe.
type CompactedEventBucket EventBucket

func (CompactedEventBucket) TableName() string { return "compacted_event_buckets" }
