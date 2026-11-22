-- Viper HUD - Feature 1: Fire Mode & Autocannon Ammo Type Indicator
-- Queries weapon fire mode and AC-8 Autocannon APHET vs FLAK state

local FireMode = {}

function FireMode.get_active_weapon_info()
    local success, info = pcall(function()
        if not rawget(_G, "Managers") or not Managers.player then return nil end
        local player = Managers.player:local_player()
        if not player or not player.player_unit then return nil end
        
        local unit = player.player_unit
        if not Unit.alive(unit) then return nil end

        local inventory_ext = Unit.has_extension(unit, "inventory_system") and Unit.extension(unit, "inventory_system")
        if not inventory_ext then return nil end

        local weapon_unit = (inventory_ext.get_weapon_unit and inventory_ext:get_weapon_unit()) 
                         or (inventory_ext.wielded_unit and inventory_ext:wielded_unit())
        if not weapon_unit or not Unit.alive(weapon_unit) then return nil end

        local weapon_ext = Unit.has_extension(weapon_unit, "weapon_system") and Unit.extension(weapon_unit, "weapon_system")
        local wielded_slot = (inventory_ext.wielded_slot_name and inventory_ext:wielded_slot_name())
        local item_data = wielded_slot and inventory_ext.get_item_data and inventory_ext:get_item_data(wielded_slot)

        -- Determine weapon name
        local weapon_name = ""
        if item_data and item_data.name then
            weapon_name = tostring(item_data.name)
        elseif weapon_unit and Unit.get_data then
            weapon_name = tostring(Unit.get_data(weapon_unit, "unit_name") or "")
        end

        local name_lower = string.lower(weapon_name)
        local is_autocannon = string.find(name_lower, "autocannon") 
                           or string.find(name_lower, "ac8") 
                           or string.find(name_lower, "ac_8")

        -- Check active ammunition / projectile mode (for Autocannon: APHET vs FLAK)
        local ammo_mode = nil
        if weapon_ext then
            if weapon_ext.current_ammo_type then
                ammo_mode = weapon_ext:current_ammo_type()
            elseif weapon_ext.ammo_type then
                ammo_mode = weapon_ext.ammo_type
            elseif weapon_ext.selected_projectile_mode then
                ammo_mode = weapon_ext:selected_projectile_mode()
            elseif weapon_ext.current_fire_mode then
                ammo_mode = weapon_ext:current_fire_mode()
            end
        end

        -- Check standard firing rate (semi vs auto)
        local fire_mode = nil
        if weapon_ext then
            if weapon_ext.current_fire_mode then
                fire_mode = weapon_ext:current_fire_mode()
            elseif weapon_ext.fire_mode then
                fire_mode = weapon_ext.fire_mode
            end
        end

        return {
            weapon_name = weapon_name,
            is_autocannon = is_autocannon,
            ammo_mode = ammo_mode,
            fire_mode = fire_mode,
        }
    end)

    if success and info then
        return info
    end
    return nil
end

function FireMode.format_display(info)
    if not info then
        return { label = "[ AUTO ]", type = "auto", is_autocannon = false }
    end

    -- Specialized Autocannon Handling: APHET vs FLAK
    if info.is_autocannon then
        local raw_ammo = string.lower(tostring(info.ammo_mode or ""))
        local rate = ""
        if info.fire_mode then
            local r = string.lower(tostring(info.fire_mode))
            if string.find(r, "auto") or string.find(r, "full") then
                rate = "AUTO"
            else
                rate = "SEMI"
            end
        end

        if string.find(raw_ammo, "flak") or string.find(raw_ammo, "airburst") or string.find(raw_ammo, "prox") then
            local label = (rate ~= "") and string.format("[ FLAK | %s ]", rate) or "[ FLAK ]"
            return { label = label, type = "flak", is_autocannon = true }
        else
            -- Default Autocannon round is APHET (AP4 armor piercing)
            local label = (rate ~= "") and string.format("[ APHET | %s ]", rate) or "[ APHET ]"
            return { label = label, type = "aphet", is_autocannon = true }
        end
    end

    -- Standard Weapons: SEMI / BURST / AUTO
    local m = string.lower(tostring(info.fire_mode or "auto"))
    if string.find(m, "semi") or string.find(m, "single") or m == "1" then
        return { label = "[ SEMI ]", type = "semi", is_autocannon = false }
    elseif string.find(m, "burst") or m == "3" then
        return { label = "[ BURST ]", type = "burst", is_autocannon = false }
    else
        return { label = "[ AUTO ]", type = "auto", is_autocannon = false }
    end
end

return FireMode
