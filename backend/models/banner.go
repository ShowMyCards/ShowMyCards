package models

// BannerSeverity controls how a Banner is styled in the UI.
// tygo:export
type BannerSeverity string

const (
	BannerSeverityInfo    BannerSeverity = "info"
	BannerSeverityWarning BannerSeverity = "warning"
	BannerSeverityError   BannerSeverity = "error"
)

// Banner is a transient, server-derived message surfaced in the UI.
//
// Banners are computed on demand (see services.BannerService) and never
// persisted: a banner exists exactly while its underlying condition is true.
// tygo:export
type Banner struct {
	// ID is stable for the lifetime of the underlying condition, so the
	// frontend can key list rendering and remember per-banner dismissals.
	ID string `json:"id"`
	// Severity controls styling.
	Severity BannerSeverity `json:"severity"`
	// Message is the full, display-ready text. The backend authors it so a new
	// banner needs no frontend change.
	Message string `json:"message"`
	// Dismissible reports whether the user may hide the banner. Dismissal state
	// is held client-side; the backend stays stateless.
	Dismissible bool `json:"dismissible"`
	// Link is an optional call to action.
	Link *BannerLink `json:"link,omitempty"`
}

// BannerLink is an optional call-to-action rendered alongside a Banner.
// tygo:export
type BannerLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}
