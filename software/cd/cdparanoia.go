package cd

// #include <cdda_paranoia.h>
import "C"

func TestIntegration() int {
	return C.PARANOIA_CB_OVERLAP
}
