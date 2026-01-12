package s3

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestEtagFileIdEquality(t *testing.T) {
	assert.False(t, eTagEqualsFileId(nil, types.FileId("f1")))
	assert.False(t, eTagEqualsFileId(aws.String("f2"), types.FileId("f1")))
	assert.False(t, eTagEqualsFileId(aws.String("f1"), types.FileId("f1")))
}

func TestStripQuotes(t *testing.T) {
	assert.Equal(t, "thing", stripQuotes(`thing`))
	assert.Equal(t, "thing", stripQuotes(`"thing`))
	assert.Equal(t, "thing", stripQuotes(`"thing"`))
}
