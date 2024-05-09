package controller

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestDetermineTransferPaths(t *testing.T) {
	t.Parallel()

	type args struct {
		sharedDir, tmpDir, name, path string
	}

	type want struct {
		destRel, destAbs, src string
	}

	type test struct {
		args args
		want want
	}

	tests := []test{
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "Name1",
				path:      "/var/source/transfer.tar.gz",
			},
			want{
				destRel: "/tmp/tmp.12345",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				src:     "/var/source/transfer.tar.gz",
			},
		},
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "Name2",
				path:      "/var/source/transfer",
			},
			want{
				destRel: "/tmp/tmp.12345/Name2",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345/Name2",
				src:     "/var/source/transfer/",
			},
		},
		{
			args{
				sharedDir: "/var/archivematica/sharedDirectory",
				tmpDir:    "/var/archivematica/sharedDirectory/tmp/tmp.12345",
				name:      "NameWithLocation",
				path:      "cae8fe7a-0ad4-495f-abf5-9d3dbd71ba36:/var/source/transfer.tar.gz",
			},
			want{
				destRel: "/tmp/tmp.12345",
				destAbs: "/var/archivematica/sharedDirectory/tmp/tmp.12345/transfer.tar.gz",
				src:     "cae8fe7a-0ad4-495f-abf5-9d3dbd71ba36:/var/source/transfer.tar.gz",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.args.name, func(t *testing.T) {
			t.Parallel()

			transferRel, filePath, path := determineTransferPaths(
				tc.args.sharedDir, tc.args.tmpDir, tc.args.name, tc.args.path,
			)

			assert.Equal(t, transferRel, tc.want.destRel, "unexpected transferRel")
			assert.Equal(t, filePath, tc.want.destAbs, "unexpected filePath")
			assert.Equal(t, path, tc.want.src, "unexpected path")
		})
	}
}

func TestMoveToInternalSharedDir(t *testing.T) {
	t.Parallel()

	tmpDir := fs.NewDir(t, "ccp",
		fs.WithDir("source",
			fs.WithDir("Images",
				fs.WithFile("MARBLES.TGA", "contents"),
			),
		),
		fs.WithDir("sharedDir",
			fs.WithDir("deposits",
				fs.WithDir("Images"),
				fs.WithDir("Images_1"),
				fs.WithDir("Images_2"),
			),
		),
	)

	dest, err := moveToInternalSharedDir(
		tmpDir.Join("sharedDir"),
		tmpDir.Join("source", "Images"),
		tmpDir.Join("sharedDir", "deposits"),
	)

	assert.NilError(t, err)
	assert.Equal(t, dest, tmpDir.Join("sharedDir", "deposits", filepath.Base(dest)))
	assert.Assert(t, fs.Equal(dest, fs.Expected(t, fs.WithFile("MARBLES.TGA", "contents"), fs.MatchAnyFileMode)))
}
