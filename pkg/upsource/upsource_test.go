package upsource

import (
	"sort"
	"testing"

	"github.com/groall/upsource-go-client/client"
)

func TestGroupReviewsByProject(t *testing.T) {
	mkReview := func(projectID string) *Review {
		return &Review{
			review: &client.ReviewDescriptorDTO{
				ReviewID: client.ReviewIdDTO{
					ProjectID: projectID,
				},
			},
		}
	}

	tests := []struct {
		name          string
		reviews       []*Review
		wantProjects  []string
		wantGroupings map[string]int // projectID -> count
	}{
		{
			name:          "empty slice",
			reviews:       []*Review{},
			wantProjects:  []string{},
			wantGroupings: map[string]int{},
		},
		{
			name:         "single review",
			reviews:      []*Review{mkReview("project1")},
			wantProjects: []string{"project1"},
			wantGroupings: map[string]int{
				"project1": 1,
			},
		},
		{
			name: "multiple reviews same project",
			reviews: []*Review{
				mkReview("project1"),
				mkReview("project1"),
				mkReview("project1"),
			},
			wantProjects: []string{"project1"},
			wantGroupings: map[string]int{
				"project1": 3,
			},
		},
		{
			name: "multiple reviews across projects",
			reviews: []*Review{
				mkReview("project2"),
				mkReview("project1"),
				mkReview("project3"),
				mkReview("project1"),
				mkReview("project2"),
			},
			wantProjects: []string{"project1", "project2", "project3"},
			wantGroupings: map[string]int{
				"project1": 2,
				"project2": 2,
				"project3": 1,
			},
		},
		{
			name: "projects sorted alphabetically",
			reviews: []*Review{
				mkReview("zebra"),
				mkReview("alpha"),
				mkReview("beta"),
			},
			wantProjects: []string{"alpha", "beta", "zebra"},
			wantGroupings: map[string]int{
				"alpha": 1,
				"beta":  1,
				"zebra": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, byProject := GroupReviewsByProject(tt.reviews)

			// Check project list
			if len(projects) != len(tt.wantProjects) {
				t.Errorf("got %d projects, want %d", len(projects), len(tt.wantProjects))
			}
			if !sort.StringsAreSorted(projects) {
				t.Error("projects are not sorted")
			}
			for i, p := range projects {
				if p != tt.wantProjects[i] {
					t.Errorf("projects[%d] = %q, want %q", i, p, tt.wantProjects[i])
				}
			}

			// Check groupings
			if len(byProject) != len(tt.wantGroupings) {
				t.Errorf("got %d project groups, want %d", len(byProject), len(tt.wantGroupings))
			}
			for projectID, wantCount := range tt.wantGroupings {
				gotCount := len(byProject[projectID])
				if gotCount != wantCount {
					t.Errorf("project %q has %d reviews, want %d", projectID, gotCount, wantCount)
				}
			}

			// Verify all reviews are accounted for
			totalReviews := 0
			for _, reviews := range byProject {
				totalReviews += len(reviews)
			}
			if totalReviews != len(tt.reviews) {
				t.Errorf("grouped %d reviews, want %d", totalReviews, len(tt.reviews))
			}
		})
	}
}

func TestGroupReviewsByProject_PreservesReviewIdentity(t *testing.T) {
	review1 := &Review{
		review: &client.ReviewDescriptorDTO{
			ReviewID: client.ReviewIdDTO{
				ProjectID: "project1",
			},
			Title: "Review 1",
		},
	}
	review2 := &Review{
		review: &client.ReviewDescriptorDTO{
			ReviewID: client.ReviewIdDTO{
				ProjectID: "project1",
			},
			Title: "Review 2",
		},
	}

	_, byProject := GroupReviewsByProject([]*Review{review1, review2})

	reviews := byProject["project1"]
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(reviews))
	}

	// Check that the reviews are the same objects
	if reviews[0] != review1 || reviews[1] != review2 {
		t.Error("reviews not preserved in correct order or identity")
	}
}
