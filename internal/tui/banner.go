package tui

// bannerInk and bannerGreen are the two halves of the CLIOne wordmark drawn
// with half-block characters — the terminal form of assets/logo.svg, two
// pixel rows of the artwork per line of text. bannerInk holds the prompt and
// "CLI", bannerGreen the "One". They are rendered side by side, one line at a
// time, so each half can carry its own color; the wordmark never mixes the
// two within a column, which is what makes a single split point per line
// enough. bannerInk is padded out to that split so the halves stay aligned.
// Editing either one means re-checking the artwork in assets/ as well.
var (
	bannerInk = []string{
		"▀█▄        ▄▀▀▀▄ █     ▀▀█▀▀ ",
		"  ▀█▄      █     █       █   ",
		" ▄█▀       █   ▄ █       █   ",
		"▀▀          ▀▀▀  ▀▀▀▀▀ ▀▀▀▀▀ ",
		"   ▀▀▀▀▀",
	}

	bannerGreen = []string{
		"▄▀▀▀▄",
		"█   █ █▄▀▀▄ ▄▀▀▀▄",
		"█   █ █   █ █▀▀▀▀",
		" ▀▀▀  ▀   ▀  ▀▀▀",
		"",
	}
)

// bannerWidth is the column width of the widest banner line, used to decide
// whether the terminal is wide enough to show the wordmark at all.
const bannerWidth = 46
