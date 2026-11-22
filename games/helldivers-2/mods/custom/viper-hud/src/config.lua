-- Viper HUD Configuration
-- Option B Layout: Tactical Weapon Wing (Bottom-Right)

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
        
        bg_color    = { r = 0.05, g = 0.08, b = 0.12, a = 0.7 }, -- Semi-transparent dark tactical panel
    },
    jammed_alert = {
        enabled = false,
        offset_y = 60,   -- Directly below center crosshair (Feature 2)
    },
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
