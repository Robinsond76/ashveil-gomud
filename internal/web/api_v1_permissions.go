package web

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// PermissionDef describes a single permission key exposed by the catalog.
// Only write and command permissions appear here; all read access is open to
// any authenticated admin/mod user.
type PermissionDef struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// allPermissions is the canonical catalog of every permission key in the
// system. It is used to power the admin UI picker and to validate PUT requests.
//
// Read access to all admin pages and GET API endpoints is open to any
// authenticated user (admin or mod). Only write operations are restricted.
var allPermissions = []PermissionDef{
	// --- In-game admin commands ---
	{Key: "badcommands", Category: "Commands", Description: "View bad/unrecognised command statistics"},
	{Key: "buff", Category: "Commands", Description: "Apply buffs to players or mobs"},
	{Key: "build", Category: "Commands", Description: "Build and link rooms"},
	{Key: "command", Category: "Commands", Description: "Execute mob commands directly"},
	{Key: "copyover", Category: "Commands", Description: "Perform a live server restart (copyover)"},
	{Key: "formset", Category: "Commands", Description: "Set form/transformation data"},
	{Key: "grant", Category: "Commands", Description: "Grant experience or other rewards to players"},
	{Key: "item", Category: "Commands", Description: "All item admin sub-commands"},
	{Key: "item.create", Category: "Commands", Description: "Create new item specifications"},
	{Key: "item.spawn", Category: "Commands", Description: "Spawn item instances in the current room"},
	{Key: "locate", Category: "Commands", Description: "Locate players or mobs anywhere in the world"},
	{Key: "mob", Category: "Commands", Description: "All mob admin sub-commands"},
	{Key: "mob.create", Category: "Commands", Description: "Create new mob specifications"},
	{Key: "mob.spawn", Category: "Commands", Description: "Spawn mob instances in the current room"},
	{Key: "modify", Category: "Commands", Description: "All modify admin sub-commands"},
	{Key: "modify.role", Category: "Commands", Description: "Change a user's role (user/mod/admin)"},
	{Key: "mute", Category: "Commands", Description: "Mute or unmute players"},
	{Key: "paz", Category: "Commands", Description: "Paz (heal) players or mobs"},
	{Key: "prepare", Category: "Commands", Description: "Pre-populate room spawn caches"},
	{Key: "questtoken", Category: "Commands", Description: "Assign or list quest tokens"},
	{Key: "rankings", Category: "Commands", Description: "View game rankings and leaderboards"},
	{Key: "redescribe", Category: "Commands", Description: "Change the description of an item in inventory"},
	{Key: "reload", Category: "Commands", Description: "Reload data files from disk"},
	{Key: "rename", Category: "Commands", Description: "Rename an item in inventory"},
	{Key: "room", Category: "Commands", Description: "All room admin sub-commands"},
	{Key: "room.copy", Category: "Commands", Description: "Copy properties between rooms"},
	{Key: "room.edit", Category: "Commands", Description: "All room edit sub-commands"},
	{Key: "room.edit.container", Category: "Commands", Description: "Edit room containers"},
	{Key: "room.edit.exits", Category: "Commands", Description: "Edit room exits"},
	{Key: "room.edit.mutators", Category: "Commands", Description: "Edit room mutators"},
	{Key: "room.info", Category: "Commands", Description: "View a room summary"},
	{Key: "room.nouns", Category: "Commands", Description: "Edit room nouns"},
	{Key: "room.set", Category: "Commands", Description: "Set room properties"},
	{Key: "room.tag", Category: "Commands", Description: "Manage room tags"},
	{Key: "server", Category: "Commands", Description: "All server admin sub-commands"},
	{Key: "skillset", Category: "Commands", Description: "Set skill levels on players"},
	{Key: "spawn", Category: "Commands", Description: "Spawn mobs or items in the room"},
	{Key: "spell", Category: "Commands", Description: "All spell admin sub-commands"},
	{Key: "spell.create", Category: "Commands", Description: "Create new spell definitions"},
	{Key: "spell.list", Category: "Commands", Description: "List all available spells"},
	{Key: "syslogs", Category: "Commands", Description: "View and follow system logs in-game"},
	{Key: "teleport", Category: "Commands", Description: "All teleport sub-commands"},
	{Key: "teleport.direction", Category: "Commands", Description: "Teleport through walls in a direction"},
	{Key: "teleport.playername", Category: "Commands", Description: "Teleport to a player by name"},
	{Key: "teleport.roomid", Category: "Commands", Description: "Teleport to a specific room ID"},
	{Key: "testarea", Category: "Commands", Description: "The admin test area: teleport in and out of a closed test zone"},
	{Key: "telemetry", Category: "Commands", Description: "Query and clear telemetry data in-game"},
	{Key: "visit", Category: "Commands", Description: "View room visit statistics"},
	{Key: "zap", Category: "Commands", Description: "Zap (damage) players or mobs"},
	{Key: "zone", Category: "Commands", Description: "All zone admin sub-commands"},

	// --- Web API write permissions ---
	{Key: "audio.write", Category: "Audio", Description: "Edit audio configuration"},
	{Key: "biomes.write", Category: "Biomes", Description: "Create, edit, and delete biomes"},
	{Key: "buffs.write", Category: "Buffs", Description: "Create, edit, and delete buffs"},
	{Key: "color-aliases.write", Category: "Colors", Description: "Create, edit, and delete color aliases"},
	{Key: "colorpatterns.write", Category: "Colors", Description: "Create, edit, and delete color patterns"},
	{Key: "config.write", Category: "Server", Description: "Edit server configuration"},
	{Key: "conversations.write", Category: "Conversations", Description: "Create, edit, and delete NPC conversations"},
	{Key: "gametime.write", Category: "GameTime", Description: "Create, edit, and delete game time calendars"},
	{Key: "items.write", Category: "Items", Description: "Create, edit, and delete items"},
	{Key: "keywords.write", Category: "Keywords", Description: "Edit keyword and alias definitions"},
	{Key: "mobs.write", Category: "Mobs", Description: "Create, edit, and delete mobs"},
	{Key: "mutators.write", Category: "Mutators", Description: "Create, edit, and delete room mutators"},
	{Key: "panels.write", Category: "Panels", Description: "Create and edit display panels"},
	{Key: "pets.write", Category: "Pets", Description: "Create, edit, and delete pets"},
	{Key: "quests.write", Category: "Quests", Description: "Create, edit, and delete quests"},
	{Key: "races.write", Category: "Races", Description: "Create, edit, and delete races"},
	{Key: "rooms.write", Category: "Rooms", Description: "Create, edit, and delete rooms"},
	{Key: "skills.write", Category: "Skills", Description: "Create, edit, and delete skills and professions"},
	{Key: "spells.write", Category: "Spells", Description: "Create, edit, and delete spells"},
	{Key: "telemetry.write", Category: "Server", Description: "Clear telemetry data"},
	{Key: "users.write", Category: "Users", Description: "Create and edit user accounts"},
	{Key: "zones.write", Category: "Zones", Description: "Create, rename, edit, and delete zones"},
}

// permissionKeySet is a fast lookup set built once at init time.
var permissionKeySet map[string]struct{}

func init() {
	permissionKeySet = make(map[string]struct{}, len(allPermissions))
	for _, p := range allPermissions {
		permissionKeySet[p.Key] = struct{}{}
	}
}

// registerModulePermission adds a module-contributed permission key to the
// catalog at runtime. Called by web.moduleAdminRegistrarImpl.RegisterPermission
// during plugin load. Safe to call multiple times with the same key (idempotent).
func registerModulePermission(key, description, category string) {
	if _, exists := permissionKeySet[key]; exists {
		return
	}
	allPermissions = append(allPermissions, PermissionDef{
		Key:         key,
		Description: description,
		Category:    category,
	})
	permissionKeySet[key] = struct{}{}
}

// isValidPermissionKey returns true when key is in the catalog.
func isValidPermissionKey(key string) bool {
	_, ok := permissionKeySet[key]
	return ok
}

// GET /admin/api/v1/permissions
func apiV1GetPermissions(w http.ResponseWriter, r *http.Request) {
	sorted := make([]PermissionDef, len(allPermissions))
	copy(sorted, allPermissions)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Category != sorted[j].Category {
			return sorted[i].Category < sorted[j].Category
		}
		return sorted[i].Key < sorted[j].Key
	})

	writeJSON(w, http.StatusOK, APIResponse[[]PermissionDef]{
		Success: true,
		Data:    sorted,
	})
}

// GET /admin/api/v1/users/{userid}/permissions
func apiV1GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	userId := resolveUserId(w, r.PathValue("userid"))
	if userId == 0 {
		return
	}

	u := loadUserRecord(w, userId)
	if u == nil {
		return
	}

	perms := u.Permissions
	if perms == nil {
		perms = []string{}
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"permissions": perms,
		},
	})
}

// PUT /admin/api/v1/users/{userid}/permissions
func apiV1PutUserPermissions(w http.ResponseWriter, r *http.Request) {
	userId := resolveUserId(w, r.PathValue("userid"))
	if userId == 0 {
		return
	}

	u := loadUserRecord(w, userId)
	if u == nil {
		return
	}

	// Prevent modifying admin accounts via this endpoint.
	if u.Role == users.RoleAdmin {
		writeAPIError(w, http.StatusForbidden, "admin permissions cannot be changed via this endpoint")
		return
	}

	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	// Validate every key against the catalog.
	for _, key := range body.Permissions {
		if !isValidPermissionKey(key) {
			writeAPIError(w, http.StatusBadRequest, "unknown permission key: "+key)
			return
		}
	}

	// Deduplicate while preserving order.
	seen := make(map[string]struct{}, len(body.Permissions))
	deduped := make([]string, 0, len(body.Permissions))
	for _, key := range body.Permissions {
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			deduped = append(deduped, key)
		}
	}

	updated := *u
	updated.Permissions = deduped

	if err := users.SaveUser(updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	users.UpdateOnlineUser(updated)

	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"permissions": deduped,
		},
	})
}
