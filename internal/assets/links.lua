---@diagnostic disable: undefined-global
function RawInline(el)
    -- Only look at LaTeX commands
    if el.format == "tex" or el.format == "latex" then
        -- Regex match for \ref{...}
        local slug = el.text:match("\\ref%{(.-)%}")

        if slug then
            -- Strategy: Check output format

            -- 1. Markdown: Convert to WikiLink [[slug]]
            if FORMAT:match("markdown") or FORMAT:match("gfm") then
                return pandoc.RawInline("markdown", "[[" .. slug .. "]]")
            end

            -- 2. HTML/Docx: Convert to bold text
            return pandoc.Strong(pandoc.Str(slug))
        end
    end
end
