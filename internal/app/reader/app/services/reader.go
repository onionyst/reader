package services

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"reader/internal/app/reader/domain"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/apperror"
)

type EditTagRequest struct {
	AddTag    string
	RemoveTag string
	EntryIDs  []int64
	UserID    int64
}

type StreamContentsData struct {
	Entries           []*models.Entry
	FeedCategoryNames map[int64]domain.FeedCategoryName
	EntryTagNames     map[int64][]string
}

type ListStreamItemIDsRequest struct {
	StreamID string
	Params   *domain.StreamParams
}

type ListStreamItemIDsResponse struct {
	IDs     []int64
	HasMore bool
}

func (s *Service) CheckToken(user *models.User, token string) bool {
	return token == s.GenerateToken(user)
}

func (s *Service) EditTag(ctx context.Context, req *EditTagRequest) error {
	if err := s.applyTag(ctx, req.AddTag, req.EntryIDs, req.UserID, true); err != nil {
		return err
	}
	return s.applyTag(ctx, req.RemoveTag, req.EntryIDs, req.UserID, false)
}

func (s *Service) GetStreamContents(ctx context.Context, entryIDs []int64, asc bool) (*StreamContentsData, error) {
	entries, err := s.repo.ListEntriesByIDs(ctx, entryIDs, asc)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("list entries: %w", err))
	}

	feedCategoryNames, err := s.repo.GetFeedAndCategoryNames(ctx)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("get feed and category names: %w", err))
	}

	resultIDs := make([]int64, len(entries))
	for i, entry := range entries {
		resultIDs[i] = entry.ID
	}

	entryTagNames, err := s.repo.GetTagNamesForEntryIDs(ctx, resultIDs)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("get tag names: %w", err))
	}

	return &StreamContentsData{
		Entries:           entries,
		FeedCategoryNames: feedCategoryNames,
		EntryTagNames:     entryTagNames,
	}, nil
}

func (s *Service) ListStreamItemIDs(ctx context.Context, req *ListStreamItemIDsRequest) (*ListStreamItemIDsResponse, error) {
	scopes, err := s.buildStreamScopes(ctx, req.StreamID)
	if err != nil {
		return nil, err
	}

	scopes = append(scopes, models.StateScope(buildStateFromFilters(req.Params.Filter, req.Params.Exclude)))

	if req.Params.StartTime != 0 {
		scopes = append(scopes, models.StartTimeScope(time.Unix(req.Params.StartTime, 0)))
	}
	if req.Params.StopTime != 0 {
		scopes = append(scopes, models.StopTimeScope(time.Unix(req.Params.StopTime, 0)))
	}
	scopes = append(scopes, models.OrderScope(req.Params.Order))
	if req.Params.Continuation != 0 {
		scopes = append(scopes, models.ContinuationScope(req.Params.Continuation, req.Params.Order))
	}

	ids, hasMore, err := s.repo.ListEntryIDs(ctx, req.Params.Count, scopes...)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("list entry IDs: %w", err))
	}

	return &ListStreamItemIDsResponse{
		IDs:     ids,
		HasMore: hasMore,
	}, nil
}

func (s *Service) ListSubscriptions(ctx context.Context) ([]*models.Category, error) {
	categories, err := s.repo.ListAllCategoriesWithFeeds(ctx)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("list all categories with feeds: %w", err))
	}
	return categories, nil
}

func (s *Service) applyTag(ctx context.Context, tag string, entryIDs []int64, userID int64, add bool) error {
	if tag == "" {
		return nil
	}

	switch tag {
	case "user/-/state/com.google/read":
		if _, err := s.repo.MarkRead(ctx, entryIDs, add); err != nil {
			return apperror.InternalServerError(fmt.Errorf("mark read=%v: %w", add, err))
		}
	case "user/-/state/com.google/starred":
		if _, err := s.repo.MarkFavorite(ctx, entryIDs, add); err != nil {
			return apperror.InternalServerError(fmt.Errorf("mark favorite=%v: %w", add, err))
		}
	default:
		tagName := extractLabelName(tag, userID)
		if tagName == "" {
			return nil
		}
		tagName = html.EscapeString(tagName)

		if add {
			return s.addLabelToEntries(ctx, tagName, entryIDs)
		}
		return s.removeLabelFromEntries(ctx, tagName, entryIDs)
	}
	return nil
}

func (s *Service) addLabelToEntries(ctx context.Context, tagName string, entryIDs []int64) error {
	tagID, found, err := s.repo.GetTagIDForName(ctx, tagName)
	if err != nil {
		return apperror.InternalServerError(fmt.Errorf("get tag ID for %s: %w", tagName, err))
	}
	if !found {
		tagID, err = s.repo.AddTag(ctx, tagName)
		if err != nil {
			return apperror.InternalServerError(fmt.Errorf("add tag %s: %w", tagName, err))
		}
	}
	if err := s.repo.AddTagForEntries(ctx, tagID, entryIDs); err != nil {
		return apperror.InternalServerError(fmt.Errorf("add tag %d for entries: %w", tagID, err))
	}
	return nil
}

func (s *Service) removeLabelFromEntries(ctx context.Context, tagName string, entryIDs []int64) error {
	tagID, found, err := s.repo.GetTagIDForName(ctx, tagName)
	if err != nil {
		return apperror.InternalServerError(fmt.Errorf("get tag ID for %s: %w", tagName, err))
	}
	if !found {
		return nil
	}
	if err := s.repo.RemoveTagForEntries(ctx, tagID, entryIDs); err != nil {
		return apperror.InternalServerError(fmt.Errorf("remove tag %d: %w", tagID, err))
	}
	return nil
}

func (s *Service) buildStreamScopes(ctx context.Context, streamID string) ([]func(*gorm.DB) *gorm.DB, error) {
	var scopes []func(*gorm.DB) *gorm.DB

	switch {
	case streamID == "user/-/state/com.google/reading-list":
		scopes = append(scopes, models.WithNormalFeeds)

	case streamID == "user/-/state/com.google/starred":
		scopes = append(scopes, models.WithNormalFeeds, models.StarredScope)

	case strings.HasPrefix(streamID, "feed/"):
		feedID, err := s.resolveFeedID(ctx, streamID[5:])
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, models.FeedScope(feedID))

	case strings.HasPrefix(streamID, "user/-/label/"):
		labelScopes, err := s.resolveLabelScopes(ctx, streamID[13:])
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, labelScopes...)
	}

	return scopes, nil
}

func (s *Service) resolveFeedID(ctx context.Context, feedRef string) (int64, error) {
	if feedRef == "" {
		return -1, nil
	}
	if id, err := strconv.ParseInt(feedRef, 10, 64); err == nil {
		return id, nil
	}
	id, found, err := s.repo.GetFeedIDForURL(ctx, feedRef)
	if err != nil {
		return 0, apperror.InternalServerError(fmt.Errorf("get feed for URL %s: %w", feedRef, err))
	}
	if !found {
		return -1, nil
	}
	return id, nil
}

func (s *Service) resolveLabelScopes(ctx context.Context, label string) ([]func(*gorm.DB) *gorm.DB, error) {
	categoryID, found, err := s.repo.GetCategoryIDForName(ctx, label)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("get category ID for name %s: %w", label, err))
	}
	if found {
		return []func(*gorm.DB) *gorm.DB{models.WithNormalFeeds, models.CategoryScope(categoryID)}, nil
	}

	tagID, found, err := s.repo.GetTagIDForName(ctx, label)
	if err != nil {
		return nil, apperror.InternalServerError(fmt.Errorf("get tag ID for name %s: %w", label, err))
	}
	if found {
		return []func(*gorm.DB) *gorm.DB{models.TagScope(tagID)}, nil
	}

	return []func(*gorm.DB) *gorm.DB{models.WithNormalFeeds}, nil
}

func buildStateFromFilters(filter, exclude string) domain.State {
	var state domain.State
	switch filter {
	case "user/-/state/com.google/read":
		state = domain.StateRead
	case "user/-/state/com.google/unread":
		state = domain.StateNotRead
	case "user/-/state/com.google/starred":
		state = domain.StateFavorite
	default:
		state = domain.StateAll
	}
	switch exclude {
	case "user/-/state/com.google/read":
		state &= domain.StateNotRead
	case "user/-/state/com.google/unread":
		state &= domain.StateRead
	case "user/-/state/com.google/starred":
		state &= domain.StateNotFavorite
	}
	return state
}

func extractLabelName(tag string, userID int64) string {
	if strings.HasPrefix(tag, "user/-/label/") {
		return tag[13:]
	}
	prefix := fmt.Sprintf("user/%d/label/", userID)
	if strings.HasPrefix(tag, prefix) {
		return tag[len(prefix):]
	}
	return ""
}
