// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package settings

// maskedSetting is a wrapper for non-reportable settings that were retrieved
// for reporting (see SetReportable and LookupForReporting).
type maskedSetting struct {
	setting NonMaskedSetting
}

var _ Setting = &maskedSetting{}

// String hides the underlying value.
func (s *maskedSetting) String(sv *Values) string {
	// Special case for non-reportable strings: we still want
	// to distinguish empty from non-empty (= customized).
	if st, ok := s.setting.(*StringSetting); ok && st.String(sv) == "" {
		return ""
	}
	return "<redacted>"
}

// Visibility returns the visibility setting for the underlying setting.
func (s *maskedSetting) Visibility() Visibility {
	return s.setting.Visibility()
}

// Key returns the key string for the underlying setting.
func (s *maskedSetting) Key() string {
	return s.setting.Key()
}

// Description returns the description string for the underlying setting.
func (s *maskedSetting) Description() string {
	return s.setting.Description()
}

// Typ returns the short (1 char) string denoting the type of setting.
func (s *maskedSetting) Typ() string {
	return s.setting.Typ()
}

// Class returns the class for the underlying setting.
func (s *maskedSetting) Class() Class {
	return s.setting.Class()
}

// TestingIsReportable is used in testing for reportability.
func TestingIsReportable(s Setting) bool {
	if _, ok := s.(*maskedSetting); ok {
		return false
	}
	if e, ok := s.(internalSetting); ok {
		return e.isReportable()
	}
	return true
}
