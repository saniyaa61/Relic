// Package core holds Relic's model and every rule from SPEC §4: time,
// progress, status, dates, search, digest and moods. It is pure Go with no
// UI or storage imports, so the app and (later) the server can share it.
//
// Where each SPEC §4 rule lives:
//
//	§2 model, ids             model.go, types.go, library.go
//	§4.1 creating an entry    entry.go (NewEntry, Library.AddEntry)
//	§4.2 status               entry.go (Entry.SetStatus)
//	§4.3 sessions             entry.go (LogSession, EditSession, DeleteSession, rewatches)
//	§4.4 progress, prompt     progress.go
//	§4.5 time spent           timespent.go (TimeEvents, TimeIndex, ConsumedByCategory)
//	§4.6 durations            format.go
//	§4.7 Your words           journey.go (Entry.YourWords)
//	§4.8 journey labels       journey.go (StartVerb, Entry.Journey)
//	§4.9 search               search.go
//	§4.10 streak              streak.go
//	§4.11 local dates         date.go
//	§4.12 digest              digest.go
//	§4.13 mood trends         moods.go
//	§4.14 favourites, Top 5   favourites.go
//	§4.15 one year ago        memory.go
//	§4.16 deleting, renaming  library.go
//
// Functions that bucket by day, month or year take the user's
// *time.Location; stored times are always UTC.
package core
