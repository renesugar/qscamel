package qingstor

import (
	"context"
	"io"
	"path"
	"strings"

	"github.com/pengsrc/go-shared/convert"
	"github.com/sirupsen/logrus"
	"github.com/yunify/qingstor-sdk-go/service"

	"github.com/yunify/qscamel/model"
)

// Reachable implement source.Reachable
func (q *QingStor) Reachable() bool {
	return true
}

// Readable implement source.Readable
func (q *QingStor) Readable() bool {
	return true
}

// List implement source.List
func (q *QingStor) List(ctx context.Context, p string) (o []model.Object, err error) {
	t, err := model.GetTask(ctx)
	if err != nil {
		logrus.Panic(err)
	}

	o = []model.Object{}
	om := make(map[string]struct{})

	// Add "/" to list specific prefix.
	cp := path.Join(t.Src.Path, p) + "/"
	// Trim left "/" to prevent object start with "/"
	cp = strings.TrimLeft(cp, "/")

	marker := ""
	first := true

	for marker != "" || first {
		resp, err := q.client.ListObjects(&service.ListObjectsInput{
			Prefix:    convert.String(cp),
			Marker:    convert.String(marker),
			Limit:     convert.Int(MaxListObjectsLimit),
			Delimiter: convert.String("/"),
		})
		if err != nil {
			return nil, err
		}
		// Both "xxx/" and "xxx" with directory content type should be treated as directory.
		// And in order to prevent duplicate job, we need to use set to filter them.
		for _, v := range resp.Keys {
			object := model.Object{
				Key:   path.Join(p, path.Base(*v.Key)),
				IsDir: *v.MimeType == DirectoryContentType,
				Size:  *v.Size,
			}

			if _, ok := om[object.Key]; !ok {
				o = append(o, object)
				om[object.Key] = struct{}{}
			}
		}
		for _, v := range resp.CommonPrefixes {
			object := model.Object{
				Key:   path.Join(p, path.Base(*v)),
				IsDir: true,
				Size:  0,
			}

			if _, ok := om[object.Key]; !ok {
				o = append(o, object)
				om[object.Key] = struct{}{}
			}
		}

		first = false
		marker = *resp.NextMarker
	}

	return
}

// Read implement source.Read
func (q *QingStor) Read(ctx context.Context, p string) (r io.Reader, err error) {
	t, err := model.GetTask(ctx)
	if err != nil {
		logrus.Panic(err)
	}

	cp := path.Join(t.Src.Path, p)
	// Trim left "/" to prevent object start with "/"
	cp = strings.TrimLeft(cp, "/")

	resp, err := q.client.GetObject(cp, nil)
	if err != nil {
		return
	}

	r = resp.Body
	return
}