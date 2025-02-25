// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build !gogit
// +build !gogit

package git

import (
	"bufio"
	"context"
	"errors"
	"path/filepath"

	"code.gitea.io/gitea/modules/log"
)

// Repository represents a Git repository.
type Repository struct {
	Path string

	tagCache *ObjectCache

	gpgSettings *GPGSettings

	batchCancel context.CancelFunc
	batchReader *bufio.Reader
	batchWriter WriteCloserError

	checkCancel context.CancelFunc
	checkReader *bufio.Reader
	checkWriter WriteCloserError

	Ctx context.Context
}

// OpenRepository opens the repository at the given path.
func OpenRepository(repoPath string) (*Repository, error) {
	return OpenRepositoryCtx(DefaultContext, repoPath)
}

// OpenRepositoryCtx opens the repository at the given path with the provided context.
func OpenRepositoryCtx(ctx context.Context, repoPath string) (*Repository, error) {
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	} else if !isDir(repoPath) {
		return nil, errors.New("no such file or directory")
	}

	repo := &Repository{
		Path:     repoPath,
		tagCache: newObjectCache(),
		Ctx:      ctx,
	}

	repo.batchWriter, repo.batchReader, repo.batchCancel = CatFileBatch(ctx, repoPath)
	repo.checkWriter, repo.checkReader, repo.checkCancel = CatFileBatchCheck(ctx, repo.Path)

	return repo, nil
}

// CatFileBatch obtains a CatFileBatch for this repository
func (repo *Repository) CatFileBatch(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.batchCancel == nil || repo.batchReader.Buffered() > 0 {
		log.Debug("Opening temporary cat file batch for: %s", repo.Path)
		return CatFileBatch(ctx, repo.Path)
	}
	return repo.batchWriter, repo.batchReader, func() {}
}

// CatFileBatchCheck obtains a CatFileBatchCheck for this repository
func (repo *Repository) CatFileBatchCheck(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.checkCancel == nil || repo.checkReader.Buffered() > 0 {
		log.Debug("Opening temporary cat file batch-check: %s", repo.Path)
		return CatFileBatchCheck(ctx, repo.Path)
	}
	return repo.checkWriter, repo.checkReader, func() {}
}

// Close this repository, in particular close the underlying gogitStorage if this is not nil
func (repo *Repository) Close() {
	if repo == nil {
		return
	}
	if repo.batchCancel != nil {
		repo.batchCancel()
		repo.batchReader = nil
		repo.batchWriter = nil
		repo.batchCancel = nil
	}
	if repo.checkCancel != nil {
		repo.checkCancel()
		repo.checkCancel = nil
		repo.checkReader = nil
		repo.checkWriter = nil
	}
}
