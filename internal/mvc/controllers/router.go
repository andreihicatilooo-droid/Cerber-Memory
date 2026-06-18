package controllers

import (
	"cerber-memory/internal/mvc/models"
	"cerber-memory/internal/pipeline"
	"fmt"
	"log"
)

// RouteItems takes the structured items from the LLM and routes them to the correct DB models.
func RouteItems(items []pipeline.ParsedItem) {
	// Map to track newly created entities in this session for linking by name
	newEntities := make(map[string]int64)

	for _, item := range items {
		var entityID int64
		var err error
		entityType := item.Type

		switch item.Type {
		case "external_resource":
			if item.URL != "" {
				entityID, err = models.AddExternalResource(item.Title, item.URL, item.Description)
				if err == nil {
					fmt.Printf("[Router] Saved Resource: %s (%s)\n", item.Title, item.URL)
				}
			}
		case "document":
			if item.Content != "" {
				entityID, err = models.AddDocument(item.Title, item.Content)
				if err == nil {
					fmt.Printf("[Router] Saved Document (ID: %d): %s\n", entityID, item.Title)
				}
			}
		case "core_memory":
			if item.Key != "" && item.Content != "" {
				entityID, err = models.UpsertCoreMemory(item.Key, item.Content, item.Category)
				if err == nil {
					fmt.Printf("[Router] Saved Core Memory: %s (Category: %s)\n", item.Key, item.Category)
				}
			}
		case "note":
			if item.Content != "" {
				entityID, err = models.AddNote(item.Title, item.Content)
				if err == nil {
					fmt.Printf("[Router] Saved Note (ID: %d): %s\n", entityID, item.Title)
				}
			}
		case "idea":
			if item.Description != "" {
				idea := models.Idea{
					Title:       item.Title,
					Description: item.Description,
					Category:    item.Category,
					Status:      item.Status,
					Priority:    item.Priority,
					IsExtracted: true,
				}
				entityID, err = models.AddIdea(idea)
				if err == nil {
					fmt.Printf("[Router] Saved Idea [%s] (ID: %d): %s\n", item.Category, entityID, item.Title)
				}
			}
		case "goal":
			if item.Title != "" {
				entityID, err = models.AddGoal(item.Title)
				if err == nil {
					fmt.Printf("[Router] Saved Goal (ID: %d): %s\n", entityID, item.Title)
				}
			}
		case "task":
			if item.Title != "" {
				task := models.Task{
					Title:           item.Title,
					Description:     item.Description,
					Priority:        item.Priority,
					Cost:            item.Cost,
					Risk:            item.Risk,
					ExpectedOutcome: item.ExpectedOutcome,
					Category:        item.Category,
					IsSubmodule:     item.IsSubmodule,
					NeedsClarify:    item.NeedsClarification,
					MissingInfo:     item.MissingInfoDetails,
				}
				for _, pst := range item.Subtasks {
					task.Subtasks = append(task.Subtasks, models.Subtask{
						Title:           pst.Title,
						Details:         pst.Details,
						Tools:           pst.Tools,
						NeedsResolution: pst.NeedsResolution,
					})
				}
				entityID, err = models.AddTask(task)
				if err == nil {
					fmt.Printf("[Router] Saved Task (ID: %d): %s\n", entityID, item.Title)
				}
			}
		case "project":
			if item.Title != "" {
				entityID, err = models.EnsureProject(item.Title, item.Workspace)
				if err == nil {
					fmt.Printf("[Router] Saved Project (ID: %d): %s\n", entityID, item.Title)
				}
			}
		default:
			log.Printf("[Router] Unknown item type: %s", item.Type)
			continue
		}

		if err != nil {
			log.Printf("[Router] Error saving %s: %v", item.Type, err)
			continue
		}

		if entityID == 0 {
			continue
		}

		// Store for potential cross-linking in the same batch
		newEntities[item.Title] = entityID
		if item.Key != "" {
			newEntities[item.Key] = entityID
		}

		// 1. Handle Links (Preserve Knowledge Connections)
		for _, link := range item.Links {
			targetID := link.TargetID
			// If targetID is a name, try to find it in the current batch
			if id, ok := newEntities[targetID]; ok {
				if err := models.LinkEntities(entityType, entityID, link.TargetType, id, link.Relation); err != nil {
					log.Printf("[Router] Error linking %s→%s: %v", entityType, link.TargetType, err)
				}
				if err := models.LinkEntities(link.TargetType, id, entityType, entityID, "related_to"); err != nil {
					log.Printf("[Router] Error back-linking %s→%s: %v", link.TargetType, entityType, err)
				}
			}

			if link.URL != "" {
				resID, _ := models.AddExternalResource("", link.URL, "")
				if resID != 0 {
					if err := models.LinkEntities(entityType, entityID, "external_resource", resID, link.Relation); err != nil {
						log.Printf("[Router] Error linking %s to resource: %v", entityType, err)
					}
					if err := models.LinkEntities("external_resource", resID, entityType, entityID, "source_for"); err != nil {
						log.Printf("[Router] Error back-linking resource to %s: %v", entityType, err)
					}
					fmt.Printf("[Router] Linked %s to URL: %s\n", entityType, link.URL)
				}
			}
		}

		// 2. Handle Project/Workspace Hierarchy
		if item.Project != "" || item.Workspace != "" {
			projID, pErr := models.EnsureProject(item.Project, item.Workspace)
			if pErr != nil {
				log.Printf("[Router] Error ensuring project %s: %v", item.Project, pErr)
			} else if projID != 0 {
				if err := models.AddToProject(projID, entityType, entityID); err != nil {
					log.Printf("[Router] Error adding %s to project: %v", entityType, err)
				} else {
					fmt.Printf("[Router] Linked %s to project: %s\n", entityType, item.Project)
				}
			}
		}

		// 3. Handle Notebooks
		if item.Notebook != "" {
			nbID, nErr := models.EnsureNotebook(item.Notebook)
			if nErr != nil {
				log.Printf("[Router] Error ensuring notebook %s: %v", item.Notebook, nErr)
			} else if nbID != 0 {
				if err := models.AddToNotebook(nbID, entityType, entityID); err != nil {
					log.Printf("[Router] Error adding %s to notebook: %v", entityType, err)
				} else {
					fmt.Printf("[Router] Added %s to notebook: %s\n", entityType, item.Notebook)
				}
			}
		}

		// 4. Handle tagging
		if len(item.Tags) > 0 {
			if err := models.AddTagsToEntity(entityType, entityID, item.Tags); err != nil {
				log.Printf("[Router] Error tagging %s: %v", entityType, err)
			} else {
				fmt.Printf("[Router] Added tags to %s: %v\n", entityType, item.Tags)
			}
		}
	}
}
