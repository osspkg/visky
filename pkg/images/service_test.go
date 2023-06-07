package images_test

import (
	"testing"

	"github.com/osspkg/visky/pkg/images"
	"github.com/stretchr/testify/require"
)

func TestImages_Resize(t *testing.T) {
	v := images.New()
	require.NoError(t, v.SetFolder("/tmp/pics"))
	info, err := v.Build("/tmp/0.png", 1000, 100)
	require.NoError(t, err)
	t.Log(info)
}
