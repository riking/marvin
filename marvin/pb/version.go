package pb


func (v *Version) Equal(other *Version) bool {
	if v == nil && other == nil {
		return true
	}
	if v == nil || other == nil {
		return false
	}
	if v.Major != other.Major {
		return false
	}
	return v.Minor == other.Minor
}

func (v *Version) Less(other *Version) bool {
	if v == nil && other == nil {
		return false
	}
	if v == nil {
		return true
	}
	if other == nil {
		return false
	}
	if v.Major < other.Major {
		return true
	}
	if v.Major > other.Major {
		return false
	}
	if v.Minor < other.Minor {
		return true
	}
	if v.Minor > other.Minor {
		return false
	}
	return false
}
