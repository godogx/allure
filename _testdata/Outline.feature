Feature: that passes

  Scenario Outline: 1 scenario 10 results
    When I pass
    And I sleep for a bit
    Then I <begin>
    And I <end>

        Examples:
            | begin | end  |
            | fail  | pass |
            | pass  | pass |
            | fail  | pass |
            | pass  | pass |
            | fail  | pass |
            | pass  | pass |
            | fail  | pass |
            | pass  | pass |
            | fail  | pass |
            | pass  | pass |

