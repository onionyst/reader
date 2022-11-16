package feeds

import (
	"reader/internal/app/reader/models"
)

// Categories
const (
	GamesCategoryName = "Games"
)

// SetupCategory setups category
func SetupCategory(name string) (int64, error) {
	categoryID, err := models.GetCategoryIDForName(name)
	if err != nil {
		return 0, err
	}

	if categoryID == -1 {
		if categoryID, err = models.AddCategory(name); err != nil {
			return 0, err
		}
	}

	return categoryID, nil
}
