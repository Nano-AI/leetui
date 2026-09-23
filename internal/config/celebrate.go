package config

// How loud an Accepted verdict is allowed to be (D-033).
//
// Three levels rather than a boolean, because "no animation" and "no acknowledgement at
// all" are different requests. Someone on a slow SSH link wants the badge without the
// frames; someone who finds the whole thing childish wants neither.

const (
	// CelebrateOff shows the verdict and nothing else.
	CelebrateOff = "off"

	// CelebrateSubtle keeps the tier badge and the figures, without motion.
	CelebrateSubtle = "subtle"

	// CelebrateFull adds the colour sweep. The default.
	CelebrateFull = "full"
)

// CelebrateLevels are the accepted values, quietest first.
func CelebrateLevels() []string {
	return []string{CelebrateOff, CelebrateSubtle, CelebrateFull}
}

// CelebrateLevel resolves the effective level, applying the default and letting
// ReduceMotion outrank it.
//
// Reduced motion demotes "full" to "subtle" rather than to "off": someone who has asked
// for no animation has not asked to stop being told they did well.
func (c Config) CelebrateLevel() string {
	level := c.UI.Celebrate
	switch level {
	case CelebrateOff, CelebrateSubtle, CelebrateFull:
	default:
		// Includes the empty string, which is what an existing config file written before
		// this setting existed will have.
		level = CelebrateFull
	}
	if level == CelebrateFull && c.UI.ReduceMotion {
		return CelebrateSubtle
	}
	return level
}
