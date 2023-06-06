package images

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImages_Resize(t *testing.T) {
	v := &Images{}
	require.NoError(t, v.SetFolder("/tmp/pics"))
	info, err := v.Build("/tmp/0.png", 1000, 100)
	require.NoError(t, err)
	t.Log(info)
}
