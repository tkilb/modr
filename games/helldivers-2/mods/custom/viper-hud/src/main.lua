-- Viper HUD - Main Entrypoint & Render Loop
-- Option B: Weapon Wing (Bottom-Right)

local Config = require("mods/tylerkilburn/viper_hud/src/config")
local FireMode = require("mods/tylerkilburn/viper_hud/src/fire_mode")

local ViperHUD = {
    version = "1.0.0",
    author = "tylerkilburn",
    enabled = true,
    Config = Config,
}

local cached_gui = nil
local screen_w, screen_h = 1920, 1080

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

    pcall(function()
        if rawget(_G, "Application") and Application.resolution then
            screen_w, screen_h = Application.resolution()
        end

        local info = FireMode.get_active_weapon_info()
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

        local gui = get_gui()
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
                local font = "core/fonts/arial"
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
    end)
end

function ViperHUD.init()
    ViperHUD.enabled = true
end

function ViperHUD.shutdown()
    ViperHUD.enabled = false
end

rawset(_G, "ViperHUD", ViperHUD)

return ViperHUD
