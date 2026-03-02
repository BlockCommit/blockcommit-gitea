/**
 * ScratchBlocks integration for SB3 diff visualization
 * This module provides scratchblocks rendering support for Scratch project diffs
 */

// Dynamic import of scratchblocks library
let scratchblocks: any = null;
let translationsLoaded = false;

/**
 * Load scratchblocks library dynamically
 */
export async function loadScratchblocks(): Promise<void> {
  if (scratchblocks) return; // Already loaded

  try {
    // Try to load from CDN first
    const script = document.createElement('script');
    script.src = 'https://cdn.jsdelivr.net/npm/scratchblocks@3.6.6/dist/scratchblocks.min.js';
    script.async = true;
    
    await new Promise((resolve, reject) => {
      script.onload = resolve;
      script.onerror = reject;
      document.head.appendChild(script);
    });

    scratchblocks = (window as any).scratchblocks;
  } catch (error) {
    console.error('Failed to load scratchblocks from CDN:', error);
    throw new Error('Could not load scratchblocks library');
  }
}

/**
 * Load translations for scratchblocks
 */
export async function loadScratchblocksTranslations(locale: string = 'en'): Promise<void> {
  if (translationsLoaded) return;

  try {
    // Load translations file
    const translationsScript = document.createElement('script');
    translationsScript.src = `https://cdn.jsdelivr.net/npm/scratchblocks@3.6.6/dist/translations-all.js`;
    translationsScript.async = true;
    
    await new Promise((resolve, reject) => {
      translationsScript.onload = resolve;
      translationsScript.onerror = reject;
      document.head.appendChild(translationsScript);
    });

    translationsLoaded = true;
  } catch (error) {
    console.warn('Failed to load scratchblocks translations:', error);
    // Continue without translations - will use English
  }
}

/**
 * Render all scratchblocks elements on the page
 */
export async function renderScratchblocks(options?: {
  style?: 'scratch2' | 'scratch3';
  languages?: string[];
  scale?: number;
}): Promise<void> {
  await loadScratchblocks();

  if (!scratchblocks) {
    console.error('Scratchblocks library not loaded');
    return;
  }

  const renderOptions = {
    style: options?.style || 'scratch3',
    languages: options?.languages || ['en', 'zh-cn'],
    scale: options?.scale || 1,
  };

  // Render block containers
  const blockElements = document.querySelectorAll('pre.blocks');
  if (blockElements.length > 0) {
    scratchblocks.renderMatching('pre.blocks', renderOptions);
  }

  // Render inline blocks
  const inlineElements = document.querySelectorAll('code.sb3-block-inline');
  if (inlineElements.length > 0) {
    scratchblocks.renderMatching('code.sb3-block-inline', {
      ...renderOptions,
      inline: true,
    });
  }
}

/**
 * Render scratchblocks in a specific container
 */
export async function renderScratchblocksInContainer(
  container: HTMLElement | string,
  options?: {
    style?: 'scratch2' | 'scratch3';
    languages?: string[];
    scale?: number;
  }
): Promise<void> {
  await loadScratchblocks();

  if (!scratchblocks) {
    console.error('Scratchblocks library not loaded');
    return;
  }

  const element = typeof container === 'string'
    ? document.querySelector(container)
    : container;

  if (!element) {
    console.error('Container not found:', container);
    return;
  }

  const renderOptions = {
    style: options?.style || 'scratch3',
    languages: options?.languages || ['en', 'zh-cn'],
    scale: options?.scale || 1,
  };

  // Find block containers within the specified container
  const blockElements = element.querySelectorAll('pre.blocks');
  if (blockElements.length > 0) {
    scratchblocks.renderMatching('pre.blocks', renderOptions);
  }

  // Find inline blocks within the specified container
  const inlineElements = element.querySelectorAll('code.sb3-block-inline');
  if (inlineElements.length > 0) {
    scratchblocks.renderMatching('code.sb3-block-inline', {
      ...renderOptions,
      inline: true,
    });
  }
}

/**
 * Initialize scratchblocks rendering for SB3 diff pages
 */
export async function initSb3DiffRenderer(): Promise<void> {
  // Wait for DOM to be ready
  if (document.readyState === 'loading') {
    await new Promise(resolve => {
      document.addEventListener('DOMContentLoaded', resolve);
    });
  }

  // Check if we're on a diff page with SB3 content
  const hasSb3Content = document.querySelector('.sb3-diff-container') !== null;
  
  if (!hasSb3Content) {
    return; // Not an SB3 diff page
  }

  try {
    // Load translations based on page language
    const htmlLang = document.documentElement.lang || 'en';
    await loadScratchblocksTranslations(htmlLang);

    // Render scratchblocks
    await renderScratchblocks({
      style: 'scratch3',
      languages: [htmlLang, 'en'],
      scale: 1,
    });

    // Set up mutation observer to handle dynamic content
    setupMutationObserver();
  } catch (error) {
    console.error('Failed to initialize SB3 diff renderer:', error);
  }
}

/**
 * Set up mutation observer to handle dynamically added scratchblocks
 */
function setupMutationObserver(): void {
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      for (const addedNode of mutation.addedNodes) {
        if (addedNode instanceof HTMLElement) {
          const hasBlocks = addedNode.querySelector('pre.blocks') !== null ||
                           addedNode.classList.contains('blocks');
          
          if (hasBlocks) {
            renderScratchblocks();
            break;
          }
        }
      }
    }
  });

  // Start observing the document
  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
}

/**
 * Parse scratchblocks syntax and return SVG string
 */
export async function parseScratchblocksToSvg(
  script: string,
  options?: {
    style?: 'scratch2' | 'scratch3';
    languages?: string[];
  }
): Promise<string> {
  await loadScratchblocks();

  if (!scratchblocks) {
    throw new Error('Scratchblocks library not loaded');
  }

  const renderOptions = {
    style: options?.style || 'scratch3',
    languages: options?.languages || ['en', 'zh-cn'],
  };

  try {
    // Parse the script
    const parsed = scratchblocks.parse(script, renderOptions);
    // Render to SVG
    const svg = scratchblocks.render(parsed, renderOptions);
    return svg;
  } catch (error) {
    console.error('Failed to parse scratchblocks:', error);
    throw error;
  }
}

/**
 * Create a preview of SB3 blocks from scratchblocks syntax
 */
export async function createBlockPreview(
  script: string,
  container: HTMLElement
): Promise<void> {
  await loadScratchblocks();

  if (!scratchblocks) {
    console.error('Scratchblocks library not loaded');
    return;
  }

  // Clear container
  container.innerHTML = '';

  // Create pre element with blocks class
  const preElement = document.createElement('pre');
  preElement.className = 'blocks';
  preElement.textContent = script;
  container.appendChild(preElement);

  // Render the blocks
  await renderScratchblocks();
}

/**
 * Export API
 */
export const ScratchBlocksAPI = {
  load: loadScratchblocks,
  render: renderScratchblocks,
  renderInContainer: renderScratchblocksInContainer,
  initDiffRenderer: initSb3DiffRenderer,
  parseToSvg: parseScratchblocksToSvg,
  createPreview: createBlockPreview,
};

// Auto-initialize when module is loaded
if (typeof window !== 'undefined') {
  // Defer initialization until after page load
  window.addEventListener('load', () => {
    initSb3DiffRenderer().catch(error => {
      console.error('Auto-initialization of scratchblocks failed:', error);
    });
  });
}