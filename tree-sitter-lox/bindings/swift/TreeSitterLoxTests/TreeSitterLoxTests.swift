import XCTest
import SwiftTreeSitter
import TreeSitterLox

final class TreeSitterLoxTests: XCTestCase {
    func testCanLoadGrammar() throws {
        let parser = Parser()
        let language = Language(language: tree_sitter_lox())
        XCTAssertNoThrow(try parser.setLanguage(language),
                         "Error loading Lox grammar")
    }
}
