#!/usr/bin/env python3
"""
Viper HUD Packager for Helldivers 2
Compiles Viper HUD into a valid Stingray binary patch archive
compliant with Chef Shared Loader and Easy Anti-Cheat.
"""

import os
import sys
import struct

def murmur64(key, seed=0):
    m = 0xc6a4a7935bd1e995
    r = 47
    data = key.encode("utf-8") if isinstance(key, str) else key
    length = len(data)
    h = seed ^ (length * m)

    n_blocks = length // 8
    for i in range(n_blocks):
        k = struct.unpack("<Q", data[i*8:(i+1)*8])[0]
        k = (k * m) & 0xffffffffffffffff
        k ^= k >> r
        k = (k * m) & 0xffffffffffffffff
        h ^= k
        h = (h * m) & 0xffffffffffffffff

    rem = length % 8
    if rem > 0:
        tail = data[n_blocks*8:]
        k = 0
        for i, b in enumerate(tail):
            k |= b << (i * 8)
        h ^= k
        h = (h * m) & 0xffffffffffffffff

    h ^= h >> r
    h = (h * m) & 0xffffffffffffffff
    h ^= h >> r
    return h & 0xffffffffffffffff

LUA_BODY = """-- HD2-Addon: mods/tylerkilburn/viper_hud
-- HD2-Requires: chef-shared-loader
-- Viper HUD - Tactical Combat UI Overlay for Helldivers 2
-- Managed by modr

local chef = ... or rawget(_G, "chef")

local ViperHUD = {
    name = "Viper HUD",
    version = "1.3.5",
    author = "tylerkilburn",
    enabled = true,
}

local state = {
    total_time = 0,
    mission_time = 0,
    last_active_weapon = nil,
    logged_mission_start = false,
    logged_active_goid = nil,
    last_eruptor_state = nil,
    logged_grip = nil,
    eruptor_cycle_start = 0,
    eruptor_was_cycling = false,
    eruptor_jammed = false,
    active_goid = nil,
    avatar_goid = nil,
}

local current_world_id = nil

-- Known weapon network IDs & default types
local WEAPONS = {
  { name = "zoaemkb", id = "15ae0889", is_eruptor=true, grip=10 },
  { name = "wg3mv1a", id = "7bfb2e06", is_ac=true, grip=22 },
  { name = "r8qx1x", id = "04645757" },
  { name = "ayy2lmj", id = "06887ef8" },
  { name = "nor4hjc", id = "0b9762b2" },
  { name = "g_b_smc", id = "0df6deff" },
  { name = "xtkwjc", id = "104184f7" },
  { name = "d3kc2ja", id = "11d8c98b" },
  { name = "vb9v6c", id = "12962fc2" },
  { name = "hsiotza", id = "15f1a9f0" },
  { name = "h9cayib", id = "17783758" },
  { name = "pyo51wa", id = "17b02218" },
  { name = "eqxvds", id = "17db8107" },
  { name = "fhqmg0a", id = "193ffddf" },
  { name = "b8ayvi", id = "1962a5a9" },
  { name = "mbrrnm", id = "1f3765c1" },
  { name = "c459gjb", id = "1fffb81f" },
  { name = "m8qm65", id = "22929ccb" },
  { name = "bhy750", id = "26acc614" },
  { name = "zjqdu0", id = "2ac9b588" },
  { name = "l4iswcb", id = "2b651283" },
  { name = "pko9xwb", id = "2cef5e3b" },
  { name = "tnc2nwa", id = "2ddd091d" },
  { name = "jq39fx", id = "2df95dfe" },
  { name = "hxz583a", id = "2f027b59" },
  { name = "h2zhqia", id = "30294946", is_sniper = true, grip = 10 },
  { name = "cltj79b", id = "303d2220" },
  { name = "yvdekca", id = "30b55bbc" },
  { name = "hg8xdoa", id = "314a1bf5" },
  { name = "na3ik9", id = "3244dc49" },
  { name = "t3doqng", id = "32fba243" },
  { name = "sdsikk", id = "33a9a66c" },
  { name = "y1uve6", id = "3478da46" },
  { name = "khs7fa", id = "3622f8dc" },
  { name = "q3otzqb", id = "366d3622" },
  { name = "gy43ise", id = "3b718900" },
  { name = "bf0m9l", id = "3bd04c18" },
  { name = "w0cy0qd", id = "3bf357e5" },
  { name = "nu2o21a", id = "3d538455" },
  { name = "s4ngu2a", id = "3d7ed060" },
  { name = "jpekgac", id = "3f87a7e8" },
  { name = "t24vbr", id = "42a6b5ba" },
  { name = "v36fp0", id = "4321fb69" },
  { name = "anefmq", id = "48643cfd" },
  { name = "vueow2b", id = "4daea8e1" },
  { name = "dtkk1fb", id = "4e87d68d", grip = 9 },
  { name = "o6pa72", id = "521229d8" },
  { name = "e11ivoc", id = "530e19df" },
  { name = "a1nulw", id = "5590232e" },
  { name = "cz7jsqa", id = "57079738" },
  { name = "oafcf7", id = "5ada6b71" },
  { name = "ho_zkj", id = "5c813e43" },
  { name = "j_9h5e", id = "5e0cd6f6" },
  { name = "fsxuztb", id = "5f3517bf" },
  { name = "xwyv6va", id = "603f3825" },
  { name = "h2u8a4d", id = "60cac78a" },
  { name = "v_6isba", id = "656d507b" },
  { name = "ug33y9f", id = "6876b30a" },
  { name = "g7cylgd", id = "69971ef9" },
  { name = "kbv6zx", id = "6b2c3bed" },
  { name = "yl2gysa", id = "6b47e064" },
  { name = "bk_010a", id = "6c26a65d" },
  { name = "kur75r", id = "6cc38f79" },
  { name = "lh9psp", id = "6ec2a850" },
  { name = "xucjhyb", id = "6f536276" },
  { name = "hbae32", id = "756a4e4d" },
  { name = "n03j_i", id = "76120872" },
  { name = "mfjhk7", id = "7dce85e3" },
  { name = "xm04dy", id = "7ea9ed9a" },
  { name = "en4ydjb", id = "82a8d3c2" },
  { name = "t2a5_nb", id = "83d1036e" },
  { name = "xjcgsqe", id = "8583e435" },
  { name = "y1lf8zc", id = "86709cb9" },
  { name = "tnfqska", id = "8696c4f0" },
  { name = "atz1va", id = "8a3c6fee" },
  { name = "pcdv3w", id = "8c0aee6e" },
  { name = "vqgjhhc", id = "8fcdd35c" },
  { name = "rq31jta", id = "903649ec" },
  { name = "bsb6dbb", id = "913cc553" },
  { name = "x_j7xb", id = "91fafc27" },
  { name = "pglrgxa", id = "942a0601" },
  { name = "wgx6kjb", id = "947ed108" },
  { name = "kzxaipa", id = "960653cb" },
  { name = "ia4hs6c", id = "967a3fed" },
  { name = "xtco_xp", id = "96c78961" },
  { name = "lp1so", id = "96c8cffd" },
  { name = "yfh87yd", id = "992ff297" },
  { name = "ub3mcka", id = "9c06f074" },
  { name = "sjviib", id = "9c2e7f43" },
  { name = "lhxfy_", id = "9ce82ac5" },
  { name = "znvy6p", id = "9e18f0cf" },
  { name = "ia44ole", id = "9f16a5a2" },
  { name = "c8bzfgk", id = "a41ddfcd" },
  { name = "gqvwvd", id = "a4eff4fa", is_sniper = true, grip = 52 },
  { name = "p640rzc", id = "a6445802" },
  { name = "cufoss", id = "a6e06a41" },
  { name = "unppkbd", id = "a9e88c96" },
  { name = "bhd28qa", id = "ab38cade" },
  { name = "n4vziwa", id = "aca565f4" },
  { name = "mxfr0g", id = "b2bfd847" },
  { name = "v6jii8j", id = "b30f446f" },
  { name = "sv_6kpa", id = "b3246bb7" },
  { name = "f00yfpg", id = "b4a17af7" },
  { name = "omhdnm", id = "b697955d", is_sniper = true, grip = 52 },
  { name = "w6vav9", id = "ba161726" },
  { name = "k04_p7e", id = "bd5f17dd" },
  { name = "njgfbx", id = "be9b793e" },
  { name = "z2j6__c", id = "bf35608c" },
  { name = "nphgvya", id = "c0e5aa7f" },
  { name = "xbn9pc", id = "c0ff5282" },
  { name = "uknvqtc", id = "c3457794" },
  { name = "nqeyoda", id = "c4235b66" },
  { name = "thqhyt", id = "c7448f32" },
  { name = "vdvv_d", id = "c7dfd9eb" },
  { name = "v7fgk0b", id = "c920c2d3" },
  { name = "ndgokac", id = "cbb5137e" },
  { name = "ptqxg5", id = "cd0e4b4a" },
  { name = "mbahpjb", id = "cfaa9ddc" },
  { name = "rdl422a", id = "d100e0c8" },
  { name = "bukir_", id = "d3c9a08c" },
  { name = "tt8fmz", id = "d496b132" },
  { name = "hbtp4i", id = "d5e0448b" },
  { name = "v9tes", id = "d724a2f9" },
  { name = "t31kcoh", id = "dd7fc64f" },
  { name = "fg4kmyl", id = "def25ee4" },
  { name = "x4eez5", id = "e1f10066" },
  { name = "flwjtq", id = "e3d29b1f" },
  { name = "kup_rdc", id = "e42d5902" },
  { name = "htx7gy", id = "e645b6ee" },
  { name = "zshdarb", id = "eb38d34e" },
  { name = "emqrkfc", id = "ee956607" },
  { name = "tdrcwja", id = "efa02c2f" },
  { name = "utnls5a", id = "f18d75e5" },
  { name = "lp5jqga", id = "f1dbc135" },
  { name = "so5lnk", id = "f2c595ea" },
  { name = "gp65z6c", id = "fb8c4465" },
  { name = "so8utv", id = "fbc7f2e1" },
  { name = "tpge9n", id = "fc1ece86" },
  { name = "e5j54r", id = "fd585726" },
  { name = "tc9txxa", id = "fde8ab3d" },
  { name = "ny71w4j", id = "fe760204" },
}

local SEED_GRIPS = {
  ["0fe87029"] = 5,  ["1220e497"] = 1,  ["15ae0889"] = 10, ["17783758"] = 52,
  ["17b02218"] = 80, ["1d3b01d7"] = 36, ["2166c7a1"] = 70, ["26acc614"] = 61,
  ["2ddd091d"] = 60, ["2df95dfe"] = 15, ["30294946"] = 10, ["314a1bf5"] = 20,
  ["3244dc49"] = 10, ["3622f8dc"] = 70, ["362577f7"] = 1,  ["366d3622"] = 20,
  ["36fcbd00"] = 70, ["3b60cae4"] = 9,  ["3d538455"] = 20, ["42a6b5ba"] = 10,
  ["437d9eed"] = 5,  ["4e87d68d"] = 9,  ["521229d8"] = 10, ["5590232e"] = 70,
  ["56e55184"] = 70, ["656d507b"] = 51, ["6b2c3bed"] = 42, ["6b47e064"] = 50,
  ["6ec2a850"] = 21, ["7bfb2e06"] = 22, ["7dce85e3"] = 15, ["82a8d3c2"] = 10,
  ["83d1036e"] = 20, ["8a3c6fee"] = 20, ["8c0aee6e"] = 51, ["8fcdd35c"] = 10,
  ["9e18f0cf"] = 51, ["9f16a5a2"] = 10, ["a4eff4fa"] = 52, ["b2bfd847"] = 20,
  ["b3246bb7"] = 50, ["b697955d"] = 52, ["ba161726"] = 14, ["c7dfd9eb"] = 36,
  ["c920c2d3"] = 1,  ["cfaa9ddc"] = 10, ["d100e0c8"] = 20, ["d496b132"] = 15,
  ["d724a2f9"] = 20, ["e645b6ee"] = 10, ["f18d75e5"] = 15, ["f94a73b3"] = 1,
  ["fd585726"] = 1,  ["feae4480"] = 1,
}

for _, w in ipairs(WEAPONS) do
  if not w.grip and SEED_GRIPS[w.id] then
    w.grip = SEED_GRIPS[w.id]
  end
end

-- Normalize ID for robust matching (handles 32-bit signed wrap and 64-bit hex)
local function norm_id(id)
    if type(id) == 'number' then
        if id < 0 then
            id = id + 4294967296
        end
        id = string.format('%x', id % 4294967296)
    elseif type(id) ~= 'string' then
        id = tostring(id or '')
    end
    id = string.gsub(string.lower(id), '^f+', '')
    return (string.gsub(id, '^0+', ''))
end

-- Role definitions: ID hash -> default call symbol name
local ROLES = {
    carry        = { id = norm_id("788b9753"), name = "n5jmvf" },
    options      = { id = norm_id("31f86165"), name = "ozvglfc" },
    fire_mode    = { id = norm_id("49c250a6"), name = "if5_pz" },
    chamber      = { id = norm_id("4a893e74"), name = "q6nx6da" },
    reload_stage = { id = norm_id("cd889dbc"), name = "r290mha" },
    held         = { id = norm_id("3132608f"), name = "jkcsuw" },
    rounds       = { id = norm_id("d7a5d63e"), name = "hhrfiqc" },
    aimpt        = { id = norm_id("bdc1cb8a"), name = "npqf7xc" },
    stratin      = { id = norm_id("95c46304"), name = "ww3bwxa" },
}

-- Dynamic field discovery cache: maps type_name -> { norm_id / fname -> (f.name or true) }
local type_fields_cache = {}
local type_cache = {}

local function get_type_fields(type_name)
    if type_fields_cache[type_name] ~= nil then
        return type_fields_cache[type_name]
    end

    local ok, info = pcall(chef.call, 'Network.object_info', type_name)
    if not ok or type(info) ~= 'table' or type(info.fields) ~= 'table' then
        type_fields_cache[type_name] = false
        return false
    end

    local field_set = {}
    for _, f in ipairs(info.fields) do
        if type(f) == 'table' then
            local fname = (type(f.name) == 'string' and f.name ~= '') and f.name or nil
            if fname then
                field_set[fname] = fname
            end
            if f.id then
                local nid = norm_id(f.id)
                if nid and nid ~= '' then
                    field_set[nid] = fname or true
                end
            end
        end
    end

    type_fields_cache[type_name] = field_set
    return field_set
end

-- Guaranteed crash-free field query: ONLY calls GameSession.game_object_field
-- if the field ID or name exists in Network.object_info(type_name)!
local function safe_field(s, goid, type_name, role_name)
    local r = ROLES[role_name]
    if not r then return nil end

    local live_set = get_type_fields(type_name)
    if not live_set then return nil end

    local fname = nil
    if r.id and live_set[r.id] then
        fname = (type(live_set[r.id]) == 'string') and live_set[r.id] or r.name
    elseif r.name and live_set[r.name] then
        fname = r.name
    end

    if not fname then return nil end

    local ok, val = pcall(chef.call, 'GameSession.game_object_field', s, goid, fname)
    if ok then
        return val
    end
    return nil
end

-- Helldiver Unit tracking for active weapon pose
local AVATAR_RESOURCE = '4d1c334d294dfa97'
local avatar_watch = nil
if chef and chef.watch then
    pcall(function()
        avatar_watch = chef.watch({ [AVATAR_RESOURCE] = true }, 2)
    end)
end

local function get_my_avatar()
    if not avatar_watch and chef and chef.watch then
        pcall(function()
            avatar_watch = chef.watch({ [AVATAR_RESOURCE] = true }, 2)
        end)
    end
    if not avatar_watch or not avatar_watch.items or #avatar_watch.items == 0 then
        return nil
    end
    if #avatar_watch.items == 1 then
        return avatar_watch.items[1]
    end
    local cam = chef.camera and chef.camera()
    if not cam then
        return avatar_watch.items[1]
    end
    local best_it, best_d = nil, nil
    for _, it in ipairs(avatar_watch.items) do
        local dx = it.x - cam.x
        local dy = it.y - cam.y
        local dz = it.z - cam.z
        local d = dx*dx + dy*dy + dz*dz
        if best_d == nil or d < best_d then
            best_it = it
            best_d = d
        end
    end
    return best_it
end

local function get_my_unit()
    local av = get_my_avatar()
    return av and av.unit or nil
end

local head_node_cache = {}
local HEAD_NAMES = { 'j_head', 'head', 'Head', 'j_Head', 'Bip01_Head', 'j_neck' }

local function get_head_node(u)
    if not u then return nil end
    if head_node_cache[u] ~= nil then
        return head_node_cache[u]
    end
    if chef and chef.unit_node then
        for _, name in ipairs(HEAD_NAMES) do
            local ok, idx = pcall(chef.unit_node, u, name)
            if ok and type(idx) == 'number' then
                head_node_cache[u] = idx
                return idx
            end
        end
    end
    head_node_cache[u] = false
    return nil
end

local function is_first_person()
    local cam = chef.camera and chef.camera()
    if not cam then return false end

    local u = get_my_unit()
    if u and chef.unit_position then
        local node = get_head_node(u)
        if node then
            local ok, hx, hy, hz = pcall(chef.unit_position, u, node)
            if ok and type(hx) == 'number' and type(hy) == 'number' and type(hz) == 'number' then
                local dx = cam.x - hx
                local dy = cam.y - hy
                local dz = cam.z - hz
                -- Camera is right at the head node in first-person scope (< 0.45m -> dist_sq < 0.20)
                -- In third-person over-the-shoulder, camera is behind shoulder (> 1.4m -> dist_sq > 1.96)
                return (dx*dx + dy*dy + dz*dz) < 0.35
            end
        end
    end

    -- Fallback if head node not resolved: horizontal camera distance to Helldiver root
    local av = get_my_avatar()
    if not av then return false end
    local dx = cam.x - av.x
    local dy = cam.y - av.y
    return (dx*dx + dy*dy) < 0.75
end

local function parse_vec3(v)
    if not v then return nil end
    if type(v) == 'table' then
        local x = v.x or v[1]
        local y = v.y or v[2]
        local z = v.z or v[3]
        if type(x) == 'number' and type(y) == 'number' and type(z) == 'number' then return x, y, z end
    end
    if type(v) == 'userdata' then
        local ok, x, y, z = pcall(chef.call, 'Vector3.to_elements', v)
        if ok and type(x) == 'number' and type(y) == 'number' and type(z) == 'number' then
            return x, y, z
        end
    end
    return nil
end

local function project_to_screen(x, y, z)
    if not chef or not chef.world_to_screen then return nil end
    local ok, sx, sy = pcall(chef.world_to_screen, x, y, z)
    if ok and type(sx) == 'number' and type(sy) == 'number' then
        return sx, sy
    end
    return nil
end

local last_aim_time = 0
local function is_player_aiming(s, av_goid)
    local raw = false
    if av_goid and s then
        local st = safe_field(s, av_goid, "vvhx8ga", "stratin")
        if st and st == 4 then
            raw = true
        end
    end
    if not raw and chef.mouse_down then
        local ok, md = pcall(chef.mouse_down, 'right')
        if ok and md == true then raw = true end
    end
    if not raw then
        local ok_pad, pad_down = pcall(function()
            local idx = chef.call('Pad1.button_index', 'left_trigger') or chef.call('Pad1.button_index', 'l2')
            if idx then
                local v = chef.call('Pad1.button', idx)
                return type(v) == 'number' and v > 0.1
            end
            return false
        end)
        if ok_pad and pad_down == true then raw = true end
    end

    if raw then
        last_aim_time = state.total_time
        return true
    end

    -- Hold state across 1-2 frame blips (e.g. during trigger pulls / bolt cycle ticks)
    if (state.total_time - last_aim_time) < 0.15 then
        return true
    end
    return false
end

local grip_index = {}
local function get_hand_grip()
    local u = get_my_unit()
    if not u or not chef.unit_alive or chef.unit_alive(u) ~= true then return nil end

    local ok_asm, has_asm = pcall(chef.call, 'Unit.has_animation_state_machine', u)
    if not ok_asm or has_asm ~= true then return nil end

    local idx = grip_index[u]
    if not idx then
        local ok_has, has_var = pcall(chef.call, 'Unit.animation_has_variable', u, 'weapon_selected')
        if not ok_has or has_var ~= true then return nil end

        local ok, find_idx = pcall(chef.call, 'Unit.animation_find_variable', u, 'weapon_selected')
        if ok and type(find_idx) == 'number' then
            idx = find_idx
            grip_index[u] = idx
        end
    end

    if idx then
        local ok, val = pcall(chef.call, 'Unit.animation_get_variable', u, idx)
        if ok and type(val) == 'number' then
            return math.floor(val + 0.5)
        end
    end
    return nil
end

local function check_world_change()
    if not chef or not chef.world_id then return end
    local wid = chef.world_id()
    if wid ~= current_world_id then
        current_world_id = wid
        type_cache = {}
        type_fields_cache = {}
        grip_index = {}
        state.last_active_weapon = nil
        state.mission_time = 0
        state.logged_mission_start = false
        state.logged_active_goid = nil
        state.eruptor_cycle_start = 0
        state.eruptor_was_cycling = false
        state.eruptor_jammed = false
        state.active_goid = nil
        state.avatar_goid = nil
        state.last_valid_grip = nil
        state.last_sway_sx = nil
        state.last_sway_sy = nil
        state.last_sway_time = 0
        head_node_cache = {}
        last_aim_time = 0
        if chef.watch then
            pcall(function()
                avatar_watch = chef.watch({ [AVATAR_RESOURCE] = true }, 2)
            end)
        end
    end
end

if chef and chef.option then
    pcall(function()
        chef.option("fire_mode_enabled", {
            type = "toggle",
            label = "Autocannon Mode (APHET / FLAK)",
            default = true,
        })
        chef.option("jam_alert_enabled", {
            type = "toggle",
            label = "Eruptor Jammed Alert",
            default = true,
        })
        chef.option("canted_reticle_enabled", {
            type = "toggle",
            label = "Sniper Canted Aim Reticle",
            default = true,
        })
        chef.option("wing_offset_x", {
            type = "slider",
            label = "Weapon Wing X Offset",
            default = -290,
            min = -800,
            max = 0,
        })
        chef.option("wing_offset_y", {
            type = "slider",
            label = "Weapon Wing Y Offset",
            default = 120,
            min = 0,
            max = 600,
        })
    end)
end

-- Active weapon query with live field validation and held weapon discrimination
local function get_active_weapon()
    if not chef or not chef.call then return nil end
    local s = chef.call('Network.game_session')
    if not s then return nil end
    local in_sess = chef.call('GameSession.in_session', s)
    if in_sess == false then return nil end

    local peer = chef.call('Network.peer_id')
    if not peer then return nil end

    local ok_objs, my_objects = pcall(chef.call, 'GameSession.objects_owned_by', s, peer)
    if not ok_objs or type(my_objects) ~= 'table' then return nil end

    local carried_weapons = {}
    local held_weapon = nil

    for _, goid in pairs(my_objects) do
        local exists = chef.call('GameSession.game_object_exists', s, goid)
        if exists == true then
            if chef.call('GameSession.game_object_is_type', s, goid, "vvhx8ga") == true then
                state.avatar_goid = goid
            end

            local spec = type_cache[goid]
            if spec == nil then
                for _, w in ipairs(WEAPONS) do
                    if chef.call('GameSession.game_object_is_type', s, goid, w.name) == true then
                        spec = w
                        type_cache[goid] = spec
                        break
                    end
                end
                if spec == nil then
                    type_cache[goid] = false
                end
            end

            if spec and spec ~= false then
                local carry = safe_field(s, goid, spec.name, "carry")
                if type(carry) == 'number' and carry % 4 == 0 then
                    local is_held = safe_field(s, goid, spec.name, "held")
                    local options = safe_field(s, goid, spec.name, "options")
                    local fire_mode = safe_field(s, goid, spec.name, "fire_mode")
                    local chamber = safe_field(s, goid, spec.name, "chamber")
                    local reload_stage = safe_field(s, goid, spec.name, "reload_stage")
                    local rounds = safe_field(s, goid, spec.name, "rounds")

                    local item = {
                        goid = goid,
                        spec = spec,
                        is_eruptor = spec.is_eruptor or false,
                        is_autocannon = spec.is_ac or false,
                        is_sniper = spec.is_sniper or false,
                        held = (is_held == true),
                        options = tonumber(options),
                        fire_mode = tonumber(fire_mode),
                        chamber = chamber,
                        reload_stage = tonumber(reload_stage) or 0,
                        rounds = tonumber(rounds),
                    }

                    carried_weapons[#carried_weapons + 1] = item
                    if item.held then
                        held_weapon = item
                    end
                end
            end
        else
            type_cache[goid] = nil
        end
    end

    if #carried_weapons == 0 then
        state.last_active_weapon = nil
        return nil
    end

    -- 1. If an object is explicitly flagged held == true, that is our active weapon!
    if held_weapon then
        state.last_active_weapon = held_weapon
        return held_weapon
    end

    -- 2. Check animation grip pose
    local grip = get_hand_grip()
    if grip and grip ~= 0 then
        for _, cw in ipairs(carried_weapons) do
            if cw.spec.grip and cw.spec.grip == grip then
                state.last_active_weapon = cw
                return cw
            end
        end
        -- Active grip is known (e.g. Talon grip 9, pistol grip 40/1, grenade)
        -- None of our known carried weapons match this grip!
        -- We must NOT falsely report that the previous sniper is still active!
        state.last_active_weapon = nil
        return nil
    end

    -- 3. While sprinting (grip == 0) or grip unknown, fallback to previously active weapon if still carried
    if state.last_active_weapon then
        for _, cw in ipairs(carried_weapons) do
            if cw.goid == state.last_active_weapon.goid then
                return cw
            end
        end
    end

    -- 4. Default to first carried weapon only if grip was completely unavailable
    if not grip then
        state.last_active_weapon = carried_weapons[1]
        return carried_weapons[1]
    end
    return nil
end

function ViperHUD.update(dt)
    if not ViperHUD.enabled or not chef then return end

    check_world_change()

    if not chef.in_mission or not chef.in_mission() then
        state.mission_time = 0
        state.logged_mission_start = false
        return
    end
    if not chef.world or not chef.world() then return end

    local delta = dt or 0.016
    state.total_time = state.total_time + delta
    state.mission_time = state.mission_time + delta

    local sw, sh = 1920, 1080
    if chef.screen then
        local w, h = chef.screen()
        if w and h and w > 0 and h > 0 then sw, sh = w, h end
    end

    if not state.logged_mission_start and chef.log then
        state.logged_mission_start = true
        chef.log(string.format("[viper_hud] Mission active. Resolution: %dx%d", sw, sh))
    end

    local px = math.max(2, math.floor(sh / 540 + 0.5))

    local TOP = chef.TOP or 2000
    local layer_bg = TOP + 100
    local layer_fg = TOP + 101

    local w = get_active_weapon()

    local weapon_swapped = false
    if w and w.goid ~= state.active_goid then
        weapon_swapped = true
        state.active_goid = w.goid
    end

    if w and w.goid ~= state.logged_active_goid and chef.log then
        state.logged_active_goid = w.goid
        chef.log(string.format("[viper_hud] Active weapon: %s (goid=%s, held=%s, fm=%s, opt=%s, chamber=%s, rl=%s)",
            tostring(w.spec and w.spec.name), tostring(w.goid), tostring(w.held),
            tostring(w.fire_mode), tostring(w.options), tostring(w.chamber), tostring(w.reload_stage)))
    end

    local grip = get_hand_grip()
    if grip and grip ~= state.logged_grip and chef.log then
        state.logged_grip = grip
        chef.log(string.format("[viper_hud] Hand grip pose: %d", grip))
    end

    -- Feature 1: Fire Mode Indicator (Restricted strictly to AC-8 Autocannon APHET / FLAK)
    local show_fire = true
    if chef.get then
        local opt = chef.get("fire_mode_enabled")
        if opt ~= nil then show_fire = opt end
    end

    if show_fire and w and w.is_autocannon and chef.rect and chef.text and chef.color then
        local label = "APHET"
        local txt_col = chef.color(255, 125, 30, 255)

        local mode_idx = math.floor((w.options or 0) / 4) % 4
        if mode_idx == 1 then
            label = "FLAK"
            txt_col = chef.color(50, 210, 255, 255)
        else
            label = "APHET"
            txt_col = chef.color(255, 125, 30, 255)
        end

        local text_w = (chef.text_width and chef.text_width(label, px)) or (px * 18)
        local pad_x = math.max(8, math.floor(px * 4))
        local panel_w = text_w + pad_x * 2

        local cap_h = px * 8.85
        local pad_y = math.max(4, math.floor(px * 2))
        local panel_h = math.floor(cap_h + pad_y * 2)

        local offset_y = (chef.get and chef.get("wing_offset_y")) or 120
        local pos_x = math.floor((sw / 2) - (panel_w / 2))
        local pos_y = offset_y
        local text_x = math.floor(pos_x + (panel_w - text_w) / 2)
        local text_y = pos_y + pad_y

        local bg_col = chef.color(10, 15, 22, 215)
        chef.rect(pos_x, pos_y, panel_w, panel_h, bg_col, layer_bg)
        chef.text(text_x, text_y, label, px, txt_col, layer_fg)
    end

    -- Feature 2: Eruptor JAM Indicator (Directly Under Center Reticle)
    local show_jam = true
    if chef.get then
        local opt = chef.get("jam_alert_enabled")
        if opt ~= nil then show_jam = opt end
    end

    if show_jam and w and w.is_eruptor and chef.rect and chef.text and chef.color then
        local has_ammo = (w.rounds ~= nil and w.rounds > 0)
        local is_bolt_needed = (w.reload_stage == 2048)

        if not has_ammo or not is_bolt_needed or state.mission_time < 2.0 then
            state.eruptor_was_cycling = false
            state.eruptor_jammed = false
        else
            if not state.eruptor_was_cycling then
                state.eruptor_was_cycling = true
                state.eruptor_cycle_start = state.total_time
                if weapon_swapped then
                    state.eruptor_jammed = true
                end
            else
                if (grip == 0) or ((state.total_time - state.eruptor_cycle_start) > 1.45) then
                    state.eruptor_jammed = true
                end
            end
        end

        local is_jammed = state.eruptor_jammed

        if chef.log then
            local key = tostring(w.reload_stage) .. ":" .. tostring(w.rounds) .. ":" .. tostring(is_jammed)
            if state.last_eruptor_state ~= key then
                state.last_eruptor_state = key
                chef.log(string.format("[viper_hud] ERUPTOR STATE: rl=%s, rounds=%s, is_jammed=%s",
                    tostring(w.reload_stage), tostring(w.rounds), tostring(is_jammed)))
            end
        end

        if is_jammed then
            local jam_text = "JAM"
            local text_w = (chef.text_width and chef.text_width(jam_text, px)) or (px * 16)
            local pad_x = math.max(8, math.floor(px * 4))
            local alert_w = text_w + pad_x * 2

            local cap_h = px * 8.85
            local pad_y = math.max(4, math.floor(px * 2))
            local alert_h = math.floor(cap_h + pad_y * 2)

            local alert_x = math.floor((sw / 2) - (alert_w / 2))
            local alert_y = math.floor((sh / 2) - 45)
            local text_x = math.floor(alert_x + (alert_w - text_w) / 2)
            local text_y = alert_y + pad_y

            local pulse = math.sin(state.total_time * 8.0)
            local alert_col = (pulse > 0) and chef.color(255, 45, 45, 245) or chef.color(255, 140, 30, 245)
            local bg_col = chef.color(25, 6, 6, 220)

            chef.rect(alert_x, alert_y, alert_w, alert_h, bg_col, layer_bg)
            chef.text(text_x, text_y, jam_text, px, alert_col, layer_fg)
        end
    end

    -- Feature 4: Canted Aim Dual Reticle for Sniper Rifles
    local show_reticle = true
    if chef.get then
        local opt = chef.get("canted_reticle_enabled")
        if opt ~= nil then show_reticle = opt end
    end

    if show_reticle and w and w.is_sniper and chef.rect and chef.color then
        local live_grip = get_hand_grip()
        if live_grip and live_grip ~= 0 then
            state.last_valid_grip = live_grip
        end
        local grip = live_grip or state.last_valid_grip
        -- Hard guard: Only display canted reticle if currently held grip matches the sniper!
        if grip and w.spec.grip and grip == w.spec.grip then
            local s = chef.call and chef.call('Network.game_session')
            local aiming = is_player_aiming(s, state.avatar_goid)
            local first_person = is_first_person()

            if chef.log and state.last_canted_aiming ~= aiming then
                state.last_canted_aiming = aiming
                local raw_pt = (aiming and state.avatar_goid) and safe_field(s, state.avatar_goid, "vvhx8ga", "aimpt") or nil
                local ax, ay, az = parse_vec3(raw_pt)
                local sx, sy = nil, nil
                if ax and ay and az then
                    sx, sy = project_to_screen(ax, ay, az)
                end
                chef.log(string.format("[viper_hud] Canted reticle: weapon=%s, grip=%s, aiming=%s, 1p=%s, av_goid=%s, raw_pt=%s, sway_screen=%s,%s",
                    tostring(w.spec and w.spec.name), tostring(grip), tostring(aiming), tostring(first_person),
                    tostring(state.avatar_goid), tostring(raw_pt ~= nil), tostring(sx), tostring(sy)))
            end

            if aiming and not first_person then
                local cx = math.floor(sw / 2)
                local cy = math.floor(sh / 2)
                local shadow_col = chef.color(0, 0, 0, 150)
                local static_col = chef.color(220, 225, 235, 175)

                -- 1. Static Camera Center Reticle (HD2 Open Circle Style + Center Dot)
                -- Smaller and reduced thickness:
                local th = math.max(1.0, px * 0.45)
                local r_static_out = math.floor(px * 3.3 + 0.5)
                local r_static_in = r_static_out - th

                if chef.ring and chef.circle then
                    chef.ring(cx, cy, math.max(0, r_static_in - 0.6), r_static_out + 0.6, shadow_col, layer_fg)
                    chef.ring(cx, cy, r_static_in, r_static_out, static_col, layer_fg + 1)
                    chef.circle(cx, cy, 1.4, shadow_col, layer_fg)
                    chef.circle(cx, cy, 0.9, static_col, layer_fg + 1)
                end

                -- 2. Dynamic Sway Reticle (Tracks physical weapon barrel aim point in 3D)
                if state.avatar_goid and chef.ring and chef.circle then
                    local raw_pt = safe_field(s, state.avatar_goid, "vvhx8ga", "aimpt")
                    local sx, sy = nil, nil
                    if raw_pt then
                        local ax, ay, az = parse_vec3(raw_pt)
                        if ax and ay and az and not (ax == 0 and ay == 0 and az == 0) then
                            sx, sy = project_to_screen(ax, ay, az)
                        end
                    end

                    -- Frame smoothing: maintain position across 1-frame raycast/projection drops
                    if sx and sy then
                        state.last_sway_sx = sx
                        state.last_sway_sy = sy
                        state.last_sway_time = state.total_time
                    elseif state.last_sway_sx and (state.total_time - (state.last_sway_time or 0)) < 0.12 then
                        sx = state.last_sway_sx
                        sy = state.last_sway_sy
                    end

                    if sx and sy then
                        local r_sway_out = math.floor(px * 1.9 + 0.5)
                        local r_sway_in = r_sway_out - th
                        local sway_col = chef.color(220, 225, 235, 185)

                        -- Outer shadow ring
                        chef.ring(sx, sy, math.max(0, r_sway_in - 0.6), r_sway_out + 0.6, shadow_col, layer_fg)
                        -- Semi-transparent grey tracking ring
                        chef.ring(sx, sy, r_sway_in, r_sway_out, sway_col, layer_fg + 1)
                        -- Center impact pip
                        chef.circle(sx, sy, 1.4, shadow_col, layer_fg)
                        chef.circle(sx, sy, 0.9, sway_col, layer_fg + 1)
                    end
                end
            end
        end
    end
end

function ViperHUD.shutdown()
    ViperHUD.enabled = false
end

function ViperHUD.on_stop()
    ViperHUD.shutdown()
end

rawset(_G, "ViperHUD", ViperHUD)
return ViperHUD
"""

def generate_bundled_lua():
    return LUA_BODY

def build_patch_archive(output_dir):
    lua_source = generate_bundled_lua()
    content_bytes = lua_source.encode("utf-8")

    magic = 0xF0000011
    types = 1
    count = 1
    head_pad = b"\x00" * (0x48 - 12)
    head = struct.pack("<III", magic, types, count) + head_pad

    LUA_TYPE = 0xa14e8dfa2cd117e2
    type_row = struct.pack("<QQQQ", 0, LUA_TYPE, 1, 0x0000001000000040)

    data_offset = 184
    payload_size = len(content_bytes)
    total_size = 8 + payload_size

    resource_name = "mods/tylerkilburn/viper_hud"
    name_hash = murmur64(resource_name)
    row = struct.pack("<QQQQQQQQQQ",
        name_hash,
        LUA_TYPE,
        data_offset,
        0, 0, 0, 0,
        total_size,
        0x0000001000000000,
        0x0000000100000040
    )

    payload_hdr = struct.pack("<II", payload_size, 2)
    archive_data = head + type_row + row + payload_hdr + content_bytes

    patch_file = os.path.join(output_dir, "9ba626afa44a3aa3.patch_0")
    gpu_file = os.path.join(output_dir, "9ba626afa44a3aa3.patch_0.gpu_resources")
    stream_file = os.path.join(output_dir, "9ba626afa44a3aa3.patch_0.stream")

    with open(patch_file, "wb") as f:
        f.write(archive_data)

    with open(gpu_file, "wb") as f:
        pass
    with open(stream_file, "wb") as f:
        pass

    print(f"[OK] Compiled {patch_file} ({len(archive_data)} bytes)")
    print(f"[OK] Generated {gpu_file} (0 bytes)")
    print(f"[OK] Generated {stream_file} (0 bytes)")

if __name__ == "__main__":
    out_dir = os.path.dirname(os.path.abspath(__file__))
    build_patch_archive(out_dir)
