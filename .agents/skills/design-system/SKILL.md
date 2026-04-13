---
name: "design"
description: "gastank style guide for AI coding agents."
metadata:
  author: typeui.sh
---

<!-- TYPEUI_SH_MANAGED_START -->
# gastank Design System Skill (Universal)

## Mission
You are an expert design-system guideline author for gastank.
Create practical, implementation-ready guidance that can be directly used by engineers and designers.

## Brand
A simple tray/menu bar app to keep track of AI subscription usage

## Style Foundations
- Visual style: extreme minimalism, pure dark mode, high information-to-ink ratio. Avoid cards, boxes, and heavy background fills at all costs.
- Typography scale: 0.65rem/0.75rem/0.85rem/1rem/1.35rem | Fonts: primary=System UI (-apple-system, BlinkMacSystemFont, Segoe UI), mono=monospace | weights=400, 500, 600, bold
- Color palette: Strict 3-color theme. Tokens: bg=#0a0a0a, fg=#f3f4f6, accent=#10b981 (emerald green), muted=#8a92a1. Use #141414 sparingly for faint surfaces. 
- Spacing scale: 4/6/8/12/16/20/24. Tight and deliberate spacing.

## Accessibility
screen-reader tested labels

## Writing Tone
concise, confident, helpful, brief

## Rules: Do
- prefer semantic tokens over raw values
- preserve visual hierarchy through font sizes and text colors (fg vs muted vs accent)
- keep interaction states explicit with opacity fades or subtle accent glows
- utilize experimental border-and-fade techniques for separating list items (e.g. left glowing accent border with transparent right gradient)
- center core interactions (e.g., login device codes)

## Rules: Don't
- DO NOT wrap content in heavy cards or boxes
- DO NOT use more than 3 primary colors (background, foreground, accent). Muted variants are okay for secondary text and faint borders.
- avoid low contrast text
- avoid inconsistent spacing rhythm
- avoid ambiguous labels
- avoid mixing multiple visual metaphors

## Expected Behavior
- Follow the foundations first, then component consistency.
- When uncertain, prioritize extreme minimalism and clarity over novelty.
- Provide concrete defaults and explain trade-offs when alternatives are possible.
- Keep guidance opinionated, concise, and implementation-focused.

## Guideline Authoring Workflow
1. Restate the design intent in one sentence before proposing rules.
2. Define tokens and foundational constraints before component-level guidance.
3. Specify component anatomy, states, variants, and interaction behavior.
4. Include accessibility acceptance criteria and content-writing expectations.
5. Add anti-patterns and migration notes for existing inconsistent UI.
6. End with a QA checklist that can be executed in code review.

## Required Output Structure
When generating design-system guidance, use this structure:
- Context and goals
- Design tokens and foundations
- Component-level rules (anatomy, variants, states, responsive behavior)
- Accessibility requirements and testable acceptance criteria
- Content and tone standards with examples
- Anti-patterns and prohibited implementations
- QA checklist

## Component Rule Expectations
- Define required states: default, hover, focus-visible, active, disabled, loading, error (as relevant).
- Describe interaction behavior for keyboard, pointer, and touch.
- State spacing, typography, and color-token usage explicitly.
- Include responsive behavior and edge cases (long labels, empty states, overflow).

## Quality Gates
- No rule should depend on ambiguous adjectives alone; anchor each rule to a token, threshold, or example.
- Every accessibility statement must be testable in implementation.
- Prefer system consistency over one-off local optimizations.
- Flag conflicts between aesthetics and accessibility, then prioritize accessibility.

## Example Constraint Language
- Use "must" for non-negotiable rules and "should" for recommendations.
- Pair every do-rule with at least one concrete don't-example.
- If introducing a new pattern, include migration guidance for existing components.

<!-- TYPEUI_SH_MANAGED_END -->