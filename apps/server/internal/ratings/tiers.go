package ratings

// Tier is a named band of ratings.
type Tier string

const (
	Bronze   Tier = "Bronze"
	Silver   Tier = "Silver"
	Gold     Tier = "Gold"
	Platinum Tier = "Platinum"
	Diamond  Tier = "Diamond"
)

// Division splits a tier below Diamond in three, III being the lowest.
type Division string

const (
	DivisionIII Division = "III"
	DivisionII  Division = "II"
	DivisionI   Division = "I"
)

// tierSpan is how many rating points each tier below Diamond spans.
const tierSpan = 200

// tierFloors are the lowest rating of each tier above Bronze, which has no
// lowest rating, in order.
var tierFloors = []struct {
	tier  Tier
	floor int
}{
	{Silver, 1100},
	{Gold, 1300},
	{Platinum, 1500},
	{Diamond, 1700},
}

var divisions = []Division{DivisionIII, DivisionII, DivisionI}

// Rank is a rating with its tier and division.
type Rank struct {
	Rating int
	Tier   Tier
	// Division is empty for Diamond.
	Division Division
}

// Label is the rank in words, such as "Gold II", or "Diamond".
func (r Rank) Label() string {
	if r.Division == "" {
		return string(r.Tier)
	}
	return string(r.Tier) + " " + string(r.Division)
}

// RankOf is the rank of a rating.
func RankOf(rating int) Rank {
	tier, floor := Bronze, tierFloors[0].floor-tierSpan
	for _, band := range tierFloors {
		if rating >= band.floor {
			tier, floor = band.tier, band.floor
		}
	}
	if tier == Diamond {
		return Rank{Rating: rating, Tier: Diamond}
	}

	// The divisions split the span evenly. Bronze has no lowest rating, so
	// below its usual span is still its lowest division.
	position := min(max((rating-floor)*len(divisions)/tierSpan, 0), len(divisions)-1)
	return Rank{Rating: rating, Tier: tier, Division: divisions[position]}
}
