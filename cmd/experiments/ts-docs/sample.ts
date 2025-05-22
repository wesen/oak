/**
 * A sample TypeScript file with various function types for testing the ts-docs tool
 */

/**
 * Adds two numbers together
 * @param a First number to add
 * @param b Second number to add
 * @returns The sum of a and b
 */
export function add(a: number, b: number): number {
  return a + b;
}

// A private utility function
function multiply(a: number, b: number): number {
  return a * b;
}

/**
 * Calculates the power of a number
 * @param base The base number
 * @param exponent The exponent to raise the base to
 * @returns The result of base^exponent
 */
export const power = (base: number, exponent: number): number => {
  let result = 1;
  for (let i = 0; i < exponent; i++) {
    result = multiply(result, base);
  }
  return result;
};

/**
 * A user in the system
 */
export class User {
  private id: string;
  public name: string;
  
  /**
   * Creates a new user
   * @param name The user's name
   */
  constructor(name: string) {
    this.name = name;
    this.id = Math.random().toString(36).substring(2, 9);
  }
  
  /**
   * Gets the user's ID
   * @returns The user's unique ID
   */
  public getId(): string {
    return this.id;
  }
  
  /**
   * Updates the user's name
   * @param newName The new name to set
   */
  public setName(newName: string): void {
    this.name = newName;
  }
}

// An arrow function with complex parameter types
export const processConfig = (config: { 
  debug: boolean;
  options: { 
    timeout: number;
    retries: number;
  }
}): string => {
  return JSON.stringify(config);
};

