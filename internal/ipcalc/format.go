package ipcalc

// splitBinaryByPrefix splits a dotted-binary IPv4 string into the network
// and host portions at the given prefix length. The "." separators stay
// where they were; the boundary "." (if any) goes with the host side so
// the network half ends on a real bit, matching the visual idiom of
// "network bits | host bits".
//
//	splitBinaryByPrefix("11000000.10101000.00000001.00000000", 24)
//	  -> ("11000000.10101000.00000001", ".00000000")
//	splitBinaryByPrefix("11000000.10101000.00000001.00000000", 26)
//	  -> ("11000000.10101000.00000001.00", "000000")
//	splitBinaryByPrefix(s, 0)  -> ("", s)
//	splitBinaryByPrefix(s, 32) -> (s, "")
func splitBinaryByPrefix(s string, prefix int) (net, host string) {
	if prefix <= 0 {
		return "", s
	}
	if prefix >= 32 {
		return s, ""
	}
	bits := 0
	for i, ch := range s {
		if ch == '.' {
			continue
		}
		bits++
		if bits == prefix {
			return s[:i+1], s[i+1:]
		}
	}
	return s, ""
}
