-- Viper HUD Configuration
-- Option B Layout: Tactical Weapon Wing (Bottom-Right) + Center Jam Alert

local Config = {
    fire_mode = {
        enabled = true,
        -- Positioned directly above vanilla ammo gauge in bottom-right corner
        offset_x = -260, -- pixels from screen right
        offset_y = 120,  -- pixels from screen bottom
        font_size = 18,
        
        -- AC-8 Autocannon Ammo Type Palettes
        color_aphet = { r = 1.0, g = 0.45, b = 0.15, a = 0.95 }, -- Heavy AP4 Orange/Gold [ APHET ]
        color_flak  = { r = 0.2, g = 0.85, b = 1.0, a = 0.95 },  -- Flak Proximity Cyan [ FLAK ]
        
        -- Standard Weapon Fire Mode Palettes
        color_auto  = { r = 1.0, g = 0.65, b = 0.2, a = 0.9 },   -- Amber [ AUTO ]
        color_burst = { r = 0.35, g = 0.85, b = 1.0, a = 0.9 }, -- Cyan [ BURST ]
        color_semi  = { r = 0.7, g = 0.95, b = 0.7, a = 0.85 },  -- Light Green [ SEMI ]
        
        bg_color    = { r = 0.05, g = 0.08, b = 0.12, a = 0.7 },
    },

    -- Feature 2: Eruptor JAMMED Alert
    -- Located directly under the center aiming reticle
    jammed_alert = {
        enabled = true,
        offset_y = 55,   -- 55 pixels below screen center (directly under crosshair)
        font_size = 19,
        text = "[ ! ] JAMMED",
        subtext = "RECHAMBER",
        color_alert = { r = 1.0, g = 0.2, b = 0.2, a = 0.95 },   -- Warning Crimson/Red
        color_flash = { r = 1.0, g = 0.5, b = 0.1, a = 0.95 },   -- Amber Flash
        bg_color    = { r = 0.15, g = 0.02, b = 0.02, a = 0.85 }, -- Dark Crimson Warning Pill
        blink_rate  = 6.0, -- Hz for alert pulse
    },

    -- Upcoming Features in Layout
    stratagem_pips = {
        enabled = false,
        offset_x = -260,
        offset_y = 155,  -- Stacked above fire mode (Feature 3)
    },
    canted_reticle = {
        enabled = false, -- Dynamic in-game crosshair tether (Feature 4)
    }
}

return Config
