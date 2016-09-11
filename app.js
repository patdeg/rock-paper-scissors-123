var myApp = angular.module('myApp', []);

myApp.controller('MyController', ['$scope', '$http', '$timeout', 
function($scope, $http, $timeout) {
	console.log(">>> MyController");

	$scope.server_play = "";
	$scope.user_play = "";

	$scope.Reset = function() {
		$scope.play_status="question";
		$scope.server_wins = 0;
		$scope.user_wins = 0;
		$scope.deuce = 0;		
		$scope.user_plays = "";
		$scope.server_plays = "";
	}
	$scope.Reset();

	$scope.RecordGame = function(winner) {
		var url = '/game?';
		url += 'id=' +  COOKIE_ID;

		$http.post(url, {
			winner: winner,
			user: $scope.user_plays,
			server: $scope.server_plays,
		})
		.success(function(data) {					
			console.log("Game recorded...")                
        })
        .error(function(errorMessage, errorCode, errorThrown) {
            console.log("Error recording game: ", errorMessage);
        });
	}

	$scope.setDelayedReset = function() {
		$timeout(function() {
			$scope.Reset();			
		}, 3000);
	}

	$scope.GetServerPlay = function() {
		console.log(">>> Play");

		var url = '/play?';
		url += 'pu=' +  $scope.user_plays;
		url += '&ps=' +  $scope.server_plays;
		console.log("Calling ", url);

		$http.get(url)
        .success(function(data) {            
            $scope.server_play = data;
            $scope.Play();
        })
        .error(function(errorMessage, errorCode, errorThrown) {
            console.log("Error - errorMessage, errorCode, errorThrown:", errorMessage, errorCode, errorThrown);                    
            alert(errorMessage);
        });
    };
    $scope.GetServerPlay();

    var iterQuestion = 0;
	$scope.setDelayedQuestion = function() {		
		iterQuestion++;
		var delay = 1500;
		if (iterQuestion<3) {
			delay = 4000;
		} else if (iterQuestion<7) {
			delay = 2500;
		}
		if ($scope.CheckIfFinish() == false) {
			$timeout(function() {
				$scope.play_status="question";
				$scope.server_play = "";
				$scope.user_play = "";
				$scope.GetServerPlay();
			}, delay);
		}
	}

	$scope.Play = function() {
		if ($scope.server_play == "") {
			console.log("Waiting server answer...");
			return
		}

		if ($scope.user_play == "") {
			console.log("Waiting user answer...");
			return
		}

		$scope.play_status="question";
		if ($scope.server_play == $scope.user_play) {
			$scope.play_status="deuce";
			$scope.deuce++;
			$scope.setDelayedQuestion();			
		} else if (
			(($scope.server_play == "rock")  && ($scope.user_play == "scissor")) ||
			(($scope.server_play == "paper")  && ($scope.user_play == "rock")) ||
			(($scope.server_play == "scissor")  && ($scope.user_play == "paper")) 
		) {
			$scope.play_status="server";
			$scope.server_wins++;
			$scope.setDelayedQuestion();
		} else if (
			(($scope.user_play == "rock")  && ($scope.server_play == "scissor")) ||
			(($scope.user_play == "paper")  && ($scope.server_play == "rock")) ||
			(($scope.user_play == "scissor")  && ($scope.server_play == "paper"))
		) {
			$scope.play_status="user";
			$scope.user_wins++;
			$scope.setDelayedQuestion();
		} else {
			$scope.play_status="unkown";
			console.log("Error: unkown game status with", $scope.user_play, $scope.server_play);
            alert("Error: unkown game status with " + $scope.user_play + " and " + $scope.server_play);
            return
		}

		var url = '/record?';
		url += 'id=' +  COOKIE_ID;
		url += '&u=' +  $scope.user_play;
		url += '&s=' +  $scope.user_play;
		url += '&pu=' +  $scope.user_plays;
		url += '&ps=' +  $scope.server_plays;
		console.log("Calling ", url);

		$http.get(url)
        .success(function(data) {
            console.log("Play recorded...")                
        })
        .error(function(errorMessage, errorCode, errorThrown) {
            console.log("Error recording play: ", errorMessage);
        });

        $scope.user_plays += $scope.user_play.charAt(0);
        $scope.server_plays += $scope.server_play.charAt(0);		

	}
	
	$scope.Rock = function() {
		console.log("Rock");
		$scope.user_play = "rock";			
		$scope.Play();
	};

	$scope.Paper = function() {
		console.log("Paper");
		$scope.user_play = "paper";			
		$scope.Play();
	};

	$scope.Scissor = function() {
		console.log("Scissor");
		$scope.user_play = "scissor";			
		$scope.Play();
	};

	$scope.CheckIfFinish = function() {
		if ($scope.server_wins+$scope.user_wins+$scope.deuce < 7) return false;
		if ($scope.server_wins == $scope.user_wins) return false;
		if ($scope.server_wins > $scope.user_wins) {
			$scope.play_status="server_won";			
			$scope.RecordGame("server");
			$scope.setDelayedReset();	
		} else {
			$scope.play_status="user_won";			
			$scope.RecordGame("user");
			$scope.setDelayedReset();	
		}
		return true;
	};
	

}]);