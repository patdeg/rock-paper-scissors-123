var myApp = angular.module('myApp', []);

myApp.controller('MyController', ['$scope', '$http', '$timeout', 
function($scope, $http, $timeout) {
	console.log(">>> MyController");
	$scope.World = "World";
	$scope.server_play = "";
	$scope.user_play = "";
	$scope.play_status="question";
	$scope.server_wins = 0;
	$scope.user_wins = 0;
	$scope.deuce = 0;
	$scope.user_plays = [];
	$scope.server_plays = [];

	$scope.GetServerPlay = function() {
		console.log(">>> Play");

		var url = '/play?';
		url += 'pu=' +  LastThree($scope.user_plays);
		url += '&ps=' +  LastThree($scope.server_plays);
		console.log("Calling ", url);

		$http.get(url)
        .success(function(data) {
            console.log("Playing: ", data);
            $scope.server_play = data;
            $scope.Play();
        })
        .error(function(errorMessage, errorCode, errorThrown) {
            console.log("Error - errorMessage, errorCode, errorThrown:", errorMessage, errorCode, errorThrown);                    
            alert(errorMessage);
        });
    };
    $scope.GetServerPlay();

	$scope.setDelayedQuestion = function() {
		$timeout(function() {
			$scope.play_status="question";
			$scope.server_play = "";
			$scope.user_play = "";
			$scope.GetServerPlay();
		}, 2000);
	}

	function LastThree(my_array) {
		var list = my_array.slice(-3);
		var result = "";
		for (var i = 0;i<list.length;i++) {
			if (result!="") result += "+";
			result += list[i];
		}
		return result
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
		url += 'u=' +  $scope.user_play;
		url += '&s=' +  $scope.user_play;
		url += '&pu=' +  LastThree($scope.user_plays);
		url += '&ps=' +  LastThree($scope.server_plays);
		console.log("Calling ", url);

		$http.get(url)
        .success(function(data) {
            console.log("Play recorded...")                
        })
        .error(function(errorMessage, errorCode, errorThrown) {
            console.log("Error recording play: ", errorMessage);
        });

        $scope.user_plays[$scope.user_plays.length] = $scope.user_play;
		$scope.server_plays[$scope.server_plays.length] = $scope.user_play;

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

	

}]);