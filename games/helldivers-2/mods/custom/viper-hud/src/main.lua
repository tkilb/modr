-- Viper HUD - Main Entrypoint & Render Loop
-- Option B: Weapon Wing (Bottom-Right)

local Config = require("mods/tylerkilburn/viper_hud/src/config")
local FireMode = require("mods/tylerkilburn/viper_hud/src/fire_mode")
local EruptorJam = require("mods/tylerkilburn/viper_hud/src/eruptor_jam")

local ViperHUD = {
    version = "1.1.0",
    author = "tylerkilburn",
    enabled = true,
    Config = Config,
}

local cached_gui = nil
local screen_w, screen_h = 1920, 1080
local total_time = 0

local function get_gui()
    if cached_gui then return cached_gui end
    if rawget(_G, "SimpleGUI") and SimpleGUI.gui then
        cached_gui = SimpleGUI.gui
        return cached_gui
    end
    if rawget(_G, "Gui") and rawget(_G, "Managers") and Managers.world then
        local world = Managers.world:world("top_ingame_view") or Managers.world:world("level_world")
        if world and rawget(_G, "World") then
            cached_gui = World.create_screen_gui(world, "immediate", "material", "core/gui/gui")
        end
    end
    return cached_gui
end

function ViperHUD.update(dt)
    if not ViperHUD.enabled then return end

    -- STRICT GUARD: Never execute during splash screen / main menu / boot!
    local Managers = rawget(_G, "Managers") or (rawget(_G, "stingray") and rawget(_G, "stingray").Managers)
    if not Managers or not Managers.player or not Managers.world then return end

    local player = (Managers.player.local_player and Managers.player:local_player())
    if not player or not player.player_unit then return end

    local Unit_ns = rawget(_G, "Unit") or (rawget(_G, "stingray") and rawget(_G, "stingray").Unit)
    if Unit_ns and Unit_ns.alive and not Unit_ns.alive(player.player_unit) then return end

    total_time = total_time + (dt or 0.016)

    pcall(function()
        if rawget(_G, "Application") and Application.resolution then
            screen_w, screen_h = Application.resolution()
        end

        local info = FireMode.get_active_weapon_info()
        local gui = get_gui()
        local font = "core/fonts/arial"

        -- 1. Feature 1: Fire Mode Indicator (Option B: Bottom-Right Weapon Wing)
        if ViperHUD.Config.fire_mode and ViperHUD.Config.fire_mode.enabled then
            local label, mode_type, is_ac = FireMode.format_display(info)
            local cfg = ViperHUD.Config.fire_mode
            local color = cfg.color_auto
            if mode_type == "aphet" then
                color = cfg.color_aphet
            elseif mode_type == "flak" then
                color = cfg.color_flak
            elseif mode_type == "semi" then
                color = cfg.color_semi
            elseif mode_type == "burst" then
                color = cfg.color_burst
            end

            local pos_x = screen_w + cfg.offset_x
            local pos_y = screen_h - cfg.offset_y
            local panel_width = is_ac and 125 or 85

            if gui and rawget(_G, "Gui") then
                if Gui.rect and cfg.bg_color then
                    local bg = cfg.bg_color
                    local bg_c = (rawget(_G, "Color4") and Color4(bg.r, bg.g, bg.b, bg.a)) 
                              or (rawget(_G, "Color") and Color(bg.a * 255, bg.r * 255, bg.g * 255, bg.b * 255))
                    if bg_c then
                        Gui.rect(gui, pos_x - 8, pos_y - 4, panel_width, 26, bg_c)
                    end
                end

                if Gui.text then
                    local txt_c = (rawget(_G, "Color4") and Color4(color.r, color.g, color.b, color.a)) 
                               or (rawget(_G, "Color") and Color(color.a * 255, color.r * 255, color.g * 255, color.b * 255))
                    local pos_vec = (rawget(_G, "Vector3") and Vector3(pos_x, pos_y, 0)) or { pos_x, pos_y, 0 }
                    if txt_c and pos_vec then
                        Gui.text(gui, label, font, cfg.font_size, pos_vec, txt_c)
                    end
                end
            elseif rawget(_G, "SimpleGUI") and SimpleGUI.text then
                SimpleGUI.text(pos_x, pos_y, label, color)
            end
        end

        -- 2. Feature 2: Eruptor JAMMED Indicator (Directly Under Center Reticle)
        if ViperHUD.Config.jammed_alert and ViperHUD.Config.jammed_alert.enabled then
            local is_jammed = EruptorJam.update(info, dt)
            if is_jammed then
                local jcfg = ViperHUD.Config.jammed_alert
                local center_x = screen_w / 2
                local center_y = (screen_h / 2) + jcfg.offset_y
                local alert_w = 135

                local pulse = math.sin(total_time * jcfg.blink_rate)
                local color = (pulse > 0) and jcfg.color_alert or jcfg.color_flash

                if gui and rawget(_G, "Gui") then
                    if Gui.rect and jcfg.bg_color then
                        local bg = jcfg.bg_color
                        local bg_c = (rawget(_G, "Color4") and Color4(bg.r, bg.g, bg.b, bg.a))
                                  or (rawget(_G, "Color") and Color(bg.a * 255, bg.r * 255, bg.g * 255, bg.b * 255))
                        if bg_c then
                            Gui.rect(gui, center_x - (alert_w / 2), center_y - 4, alert_w, 28, bg_c)
                        end
                    end

                    if Gui.text then
                        local txt_c = (rawget(_G, "Color4") and Color4(color.r, color.g, color.b, color.a))
                                   or (rawget(_G, "Color") and Color(color.a * 255, color.r * 255, color.g * 255, color.b * 255))
                        local pos_vec = (rawget(_G, "Vector3") and Vector3(center_x - (alert_w / 2) + 10, center_y, 0)) or { center_x - (alert_w / 2) + 10, center_y, 0 }
                        if txt_c and pos_vec then
                            Gui.text(gui, jcfg.text, font, jcfg.font_size, pos_vec, txt_c)
                        end
                    end
                elseif rawget(_G, "SimpleGUI") and SimpleGUI.text then
                    SimpleGUI.text(center_x - (alert_w / 2), center_y, jcfg.text, color)
                end
            end
        end
    end)
end

function ViperHUD.init()
    ViperHUD.enabled = true
end

function ViperHUD.shutdown()
    ViperHUD.enabled = false
end

-- Boot Mod Loader / MDL Lifecycle Hooks
function ViperHUD.on_enable(ctx)
    ViperHUD.init()
end

function ViperHUD.on_update(ctx, dt)
    ViperHUD.update(dt)
end

function ViperHUD.on_disable(ctx)
    ViperHUD.shutdown()
end

rawset(_G, "ViperHUD", ViperHUD)

return ViperHUD
