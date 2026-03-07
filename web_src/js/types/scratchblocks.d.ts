/**
 * Type definitions for scratchblocks library
 */

declare module 'scratchblocks' {
  export interface ParseOptions {
    style?: 'scratch2' | 'scratch3';
    languages?: string[];
  }

  export interface RenderOptions extends ParseOptions {
    scale?: number;
    inline?: boolean;
  }

  export interface ParsedBlock {
    // Simplified type for parsed scratchblocks
    [key: string]: any;
  }

  export function parse(script: string, options?: ParseOptions): ParsedBlock;
  export function render(parsed: ParsedBlock, options?: RenderOptions): string;
  export function renderMatching(selector: string, options?: RenderOptions): void;

  const scratchblocks: {
    parse: typeof parse;
    render: typeof render;
    renderMatching: typeof renderMatching;
  };

  export default scratchblocks;
}