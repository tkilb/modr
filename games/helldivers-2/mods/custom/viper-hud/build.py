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
    version = "1.2.3",
    author = "tylerkilburn",
    enabled = true,
}

local state = {
    total_time = 0,
    mission_time = 0,
    last_active_weapon = nil,
    logged_mission_start = false,
    logged_active_goid = nil,
}

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
  { name = "h2zhqia", id = "30294946" },
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
  { name = "gqvwvd", id = "a4eff4fa" },
  { name = "p640rzc", id = "a6445802" },
  { name = "cufoss", id = "a6e06a41" },
  { name = "unppkbd", id = "a9e88c96" },
  { name = "bhd28qa", id = "ab38cade" },
  { name = "n4vziwa", id = "aca565f4" },
  { name = "mxfr0g", id = "b2bfd847" },
  { name = "v6jii8j", id = "b30f446f" },
  { name = "sv_6kpa", id = "b3246bb7" },
  { name = "f00yfpg", id = "b4a17af7" },
  { name = "omhdnm", id = "b697955d" },
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

-- Normalize ID for robust matching
local function norm_id(id)
    if type(id) == 'number' then
        id = string.format('%x', id)
    elseif type(id) ~= 'string' then
        id = tostring(id or '')
    end
    return (string.gsub(string.lower(id), '^0+', ''))
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
}

-- Dynamic field discovery cache: maps type_name -> { norm_id -> (f.name or true) }
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
        if type(f) == 'table' and f.id then
            local nid = norm_id(f.id)
            if nid then
                field_set[nid] = (type(f.name) == 'string' and f.name ~= '') and f.name or true
            end
        end
    end

    type_fields_cache[type_name] = field_set
    return field_set
end

-- Guaranteed crash-free field query: ONLY calls GameSession.game_object_field
-- if the field ID exists in Network.object_info(type_name)!
local function safe_field(s, goid, type_name, role_name)
    local r = ROLES[role_name]
    if not r then return nil end

    local live_set = get_type_fields(type_name)
    if not live_set or not live_set[r.id] then
        -- This field does not exist on this network type in the running build!
        return nil
    end

    local fname = (type(live_set[r.id]) == 'string' and live_set[r.id] ~= '') and live_set[r.id] or r.name
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

local function get_my_unit()
    if not avatar_watch or not avatar_watch.items or #avatar_watch.items == 0 then
        return nil
    end
    if #avatar_watch.items == 1 then
        return avatar_watch.items[1].unit
    end
    local cam = chef.camera and chef.camera()
    if not cam then
        return avatar_watch.items[1].unit
    end
    local best_u, best_d = nil, nil
    for _, it in ipairs(avatar_watch.items) do
        local dx = it.x - cam.x
        local dy = it.y - cam.y
        local dz = it.z - cam.z
        local d = dx*dx + dy*dy + dz*dz
        if best_d == nil or d < best_d then
            best_u = it.unit
            best_d = d
        end
    end
    return best_u
end

local grip_var_idx = nil
local function get_hand_grip()
    local u = get_my_unit()
    if not u or not chef.unit_alive or chef.unit_alive(u) ~= true then return nil end
    if not grip_var_idx then
        local ok, idx = pcall(chef.call, 'Unit.animation_find_variable', u, 'weapon_selected')
        if ok and type(idx) == 'number' then
            grip_var_idx = idx
        end
    end
    if grip_var_idx then
        local ok, val = pcall(chef.call, 'Unit.animation_get_variable', u, grip_var_idx)
        if ok and type(val) == 'number' then
            return math.floor(val + 0.5)
        end
    end
    return nil
end

if chef and chef.option then
    pcall(function()
        chef.option("fire_mode_enabled", {
            type = "toggle",
            label = "Show Fire Mode (Option B)",
            default = true,
        })
        chef.option("jam_alert_enabled", {
            type = "toggle",
            label = "Eruptor Jammed Alert",
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

    -- 2. If held == true wasn't found (e.g. during weapon switch transition), check animation grip pose
    local grip = get_hand_grip()
    if grip then
        for _, cw in ipairs(carried_weapons) do
            if cw.spec.grip and cw.spec.grip == grip then
                state.last_active_weapon = cw
                return cw
            end
        end
    end

    -- 3. Fallback to previously active weapon if it is still carried
    if state.last_active_weapon then
        for _, cw in ipairs(carried_weapons) do
            if cw.goid == state.last_active_weapon.goid then
                state.last_active_weapon = cw
                return cw
            end
        end
    end

    -- 4. Default to first carried weapon
    state.last_active_weapon = carried_weapons[1]
    return carried_weapons[1]
end

function ViperHUD.update(dt)
    if not ViperHUD.enabled or not chef then return end

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

    if w and w.goid ~= state.logged_active_goid and chef.log then
        state.logged_active_goid = w.goid
        chef.log(string.format("[viper_hud] Active weapon: %s (goid=%s, held=%s, fm=%s, opt=%s, chamber=%s, rl=%s)",
            tostring(w.spec and w.spec.name), tostring(w.goid), tostring(w.held),
            tostring(w.fire_mode), tostring(w.options), tostring(w.chamber), tostring(w.reload_stage)))
    end

    -- Feature 1: Fire Mode Indicator (Option B: Bottom-Right Weapon Wing)
    local show_fire = true
    if chef.get then
        local opt = chef.get("fire_mode_enabled")
        if opt ~= nil then show_fire = opt end
    end

    if show_fire and w and chef.rect and chef.text and chef.color then
        local offset_x = (chef.get and chef.get("wing_offset_x")) or -290
        local offset_y = (chef.get and chef.get("wing_offset_y")) or 120

        local pos_x = sw + offset_x
        local pos_y = offset_y

        local label = "[ AUTO ]"
        local txt_col = chef.color(255, 170, 40, 240)

        if w.is_autocannon then
            local mode_idx = math.floor((w.options or 0) / 4) % 4
            if mode_idx == 1 then
                label = "[ FLAK ]"
                txt_col = chef.color(50, 210, 255, 255)
            else
                label = "[ APHET ]"
                txt_col = chef.color(255, 125, 30, 255)
            end
        elseif w.is_eruptor then
            label = "[ BOLT ]"
            txt_col = chef.color(255, 180, 50, 240)
        else
            local fm = (w.fire_mode or 0) % 8
            if fm == 2 then
                label = "[ SEMI ]"
                txt_col = chef.color(140, 240, 140, 240)
            elseif fm == 3 then
                label = "[ BURST ]"
                txt_col = chef.color(80, 200, 255, 240)
            elseif fm == 4 then
                label = "[ VOLLEY ]"
                txt_col = chef.color(200, 120, 255, 240)
            elseif fm == 5 then
                label = "[ SAFE ]"
                txt_col = chef.color(100, 240, 100, 240)
            elseif fm == 6 then
                label = "[ UNSAFE ]"
                txt_col = chef.color(255, 80, 80, 240)
            else
                label = "[ AUTO ]"
                txt_col = chef.color(255, 170, 40, 240)
            end
        end

        local text_w = (chef.text_width and chef.text_width(label, px)) or 75
        local panel_w = text_w + 24
        local panel_h = 24

        local bg_col = chef.color(10, 15, 22, 215)
        chef.rect(pos_x, pos_y, panel_w, panel_h, bg_col, layer_bg)
        chef.text(pos_x + 12, pos_y + 4, label, px, txt_col, layer_fg)
    end

    -- Feature 2: Eruptor JAMMED Indicator (Directly Under Center Reticle)
    local show_jam = true
    if chef.get then
        local opt = chef.get("jam_alert_enabled")
        if opt ~= nil then show_jam = opt end
    end

    if show_jam and w and w.is_eruptor and chef.rect and chef.text and chef.color then
        local in_hand = (w.held == true) or (get_hand_grip() == 10)
        local is_cycling = (w.reload_stage ~= 0)
        local has_ammo = (w.rounds == nil or w.rounds > 0)
        local is_jammed = in_hand and (w.chamber == false and not is_cycling and has_ammo and state.mission_time > 2.0)

        if is_jammed then
            local jam_text = "[ ! ] JAMMED"
            local text_w = (chef.text_width and chef.text_width(jam_text, px)) or 95
            local alert_w = text_w + 24
            local alert_h = 24
            local alert_x = (sw / 2) - (alert_w / 2)
            local alert_y = (sh / 2) - 45

            local pulse = math.sin(state.total_time * 8.0)
            local alert_col = (pulse > 0) and chef.color(255, 45, 45, 245) or chef.color(255, 140, 30, 245)
            local bg_col = chef.color(25, 6, 6, 220)

            chef.rect(alert_x, alert_y, alert_w, alert_h, bg_col, layer_bg)
            chef.text(alert_x + 12, alert_y + 4, jam_text, px, alert_col, layer_fg)
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
